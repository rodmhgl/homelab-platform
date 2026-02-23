# Building Interactive CLI Wizards with Bubbletea: From Flags to TUI

**Date:** February 23, 2026
**Author:** Rod Stewart (with Claude Sonnet 4.5)
**Project:** AKS Home Lab Internal Developer Platform
**Task:** Implementing `rdp infra create storage` (#69) and `rdp infra create vault` (#70)

## The Task

Replace manual YAML authoring and Git commits with guided, interactive `rdp infra create` commands. Developers should be able to provision Azure Storage Accounts and Key Vaults by answering questions, not by editing Crossplane Claim manifests.

The interesting part wasn't the TUI rendering. It was figuring out how to map a sequential form wizard onto bubbletea's Elm Architecture — and deciding what to validate where.

## Why Bubbletea, Not Flags

The obvious approach is flags:

```bash
rdp infra create storage --name my-bucket --namespace default --location southcentralus --tier Standard --redundancy LRS
```

This works for scripts and CI. But for interactive use, it's hostile — the user needs to know every flag name, every valid value, and the correct format before running the command. Get one wrong and you start over.

A sequential TUI wizard solves this by **decomposing one complex decision into many simple ones**. Each step:

- Shows only what's relevant now
- Validates immediately (not after 30 seconds of API round-trip)
- Offers selection lists where free-text entry would be error-prone

The tradeoff: bubbletea adds three dependencies and ~1200 lines of code. For two commands. That's a real cost — but it's amortized across every future interactive command (`rdp scaffold create`, `rdp secrets create`, etc.) because the shared TUI components are reusable.

## The Elm Architecture Applied to Forms

Bubbletea follows the Elm Architecture: `Model → Update → View`. Your model holds all state, `Update` receives messages and returns the next model, `View` renders the current model. No side effects in View, no state mutation outside Update.

For a sequential form, the model is a state machine:

```go
type StorageModel struct {
    state string // "welcome", "inputName", "inputNamespace", ..., "submitting", "success", "error"
    // ... collected field values
}
```

The state machine has 12 states. Each `Update` call either:
1. Validates input and advances to the next state
2. Shows an inline error and stays in the current state
3. Fires an async command (API call) and transitions to "submitting"

This looks simple on paper. In practice, three things caught me.

### Gotcha 1: Selection Lists Need Separate Input Modes

Text fields and selection lists have fundamentally different input handling. Text fields consume all keypresses. Selection lists need arrow keys for navigation and Enter for selection.

The naive approach — one `textinput.Model` shared across all states — breaks when you hit a selection state. The text input captures `up`/`down` as cursor movement, not list navigation.

The fix is simple but non-obvious: only route keypresses to `textinput.Update` when the current state is a text input state. Selection states handle `up`/`down`/`enter` directly:

```go
case "up":
    if m.state == "inputLocation" && m.cursor > 0 {
        m.cursor--
        return m, nil
    }
    if m.state == "inputTier" && m.cursor > 0 {
        m.cursor--
        return m, nil
    }
```

You could abstract this into a generic `SelectModel`, but for two commands with 2-3 selection fields each, the direct approach is clearer. Premature abstraction is the enemy of readable state machines.

### Gotcha 2: Y/N States vs Confirmation States

The storage wizard has two Y/N prompts: versioning toggle and final confirmation. Both use `y`/`n` keypresses, but they need different behavior:

- **Versioning:** Y sets `enableVersioning = true`, advances to next field
- **Confirmation:** Y fires the API call, N quits the program

If you handle all `y`/`n` in a single switch case, you'll accidentally fire API calls when the user is just toggling versioning. The state check must come first:

```go
case "y", "Y":
    if m.state == "confirmation" {
        m.state = "submitting"
        return m, m.submitClaim()
    }
    if m.state == "inputVersioning" {
        m.enableVersioning = true
        m.state = "inputRepoOwner"
        return m, nil
    }
```

Order matters. If confirmation is checked first, a `y` during the versioning step won't accidentally trigger submission.

### Gotcha 3: View Logic Gets Messy Fast

The View function needs to show all *completed* fields above the current input. This sounds simple — just check `m.name != ""`. But it gets tangled because the namespace field has a default value of `"default"`, so it's never empty. And field visibility depends on *which state you're past*, not just whether the value is set.

The result is a wall of state-checking conditionals:

```go
if m.state != "welcome" && m.state != "inputName" && m.state != "inputNamespace" && m.state != "inputLocation" && m.location != "" {
    b.WriteString(RenderFieldRow("Location", m.location))
}
```

This is ugly but correct. An alternative is an ordered list of `(state, label, value)` tuples with a "show if past this state" predicate. That's cleaner for 10+ fields but overkill for 6-8. The threshold for refactoring is the next wizard that has more fields.

## Git URL Parsing: Two Formats, Many Edge Cases

The commands auto-detect the Git repository from `git remote get-url origin`. This saves the user from typing `--repo-owner rodmhgl --repo-name my-app` every time. But Git remotes come in multiple formats:

```text
SSH:   git@github.com:rodmhgl/my-app.git
HTTPS: https://github.com/rodmhgl/my-app.git
```

Parsing both correctly requires handling:

- `.git` suffix (optional in both formats)
- SSH colon separator (`:`) vs HTTPS path separator (`/`)
- `http://` vs `https://` (some corporate proxies still use HTTP)

```go
func ParseGitURL(url string) (*GitRepo, error) {
    url = strings.TrimSuffix(url, ".git")

    if strings.HasPrefix(url, "git@") {
        // SSH: split on ":", take second part, split on "/"
        parts := strings.Split(url, ":")
        ownerRepo := strings.Split(parts[1], "/")
        return &GitRepo{Owner: ownerRepo[0], Name: ownerRepo[1]}, nil
    }

    if strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "http://") {
        // HTTPS: split on "/", owner is [3], repo is [4]
        parts := strings.Split(url, "/")
        return &GitRepo{Owner: parts[3], Name: parts[4]}, nil
    }

    return nil, fmt.Errorf("unsupported URL format: %s", url)
}
```

The important design decision: **parse errors are not fatal**. If we can't detect the repo, we prompt the user for manual entry. This graceful degradation means the command works:

- In a Git repository with SSH remote (auto-fill)
- In a Git repository with HTTPS remote (auto-fill)
- Outside any Git repository (manual entry)
- In a repo with a weird remote URL format (manual entry)

The fallback path is always there.

## Validation Alignment: CLI, API, and Gatekeeper

Three validation layers enforce the same rules independently:

| Rule | CLI (pre-submit) | API (request validation) | Gatekeeper (admission) |
| --- | --- | --- | --- |
| DNS label format | `ValidateDNSLabel()` regex | Request struct validation | N/A (K8s naming) |
| Location whitelist | Only `southcentralus`, `eastus2` in selection list | 400 Bad Request | `CrossplaneClaimLocation` constraint |
| No public access | Always `false`, never exposed to user | Request struct default | `CrossplaneNoPublicAccess` constraint |
| Retention days 7-90 | `ValidateRetentionDays()` range check | 400 Bad Request | N/A (XRD schema) |

The CLI validation is a **fast feedback loop** — the user sees errors inline, immediately, without waiting for an API round-trip. But it's not the only layer. The API validates again (someone might use `curl`), and Gatekeeper validates again (someone might `kubectl apply` directly).

This defense-in-depth means the CLI can be a bit more relaxed — if a validation rule changes in the API, the CLI won't block users from using the new value. The worst case is the CLI allows something the API rejects, and the user sees an error from the API. That's acceptable.

The one non-negotiable: `publicAccess` is **never exposed to the user**. It's hardcoded to `false`. The Gatekeeper constraint would reject `true` anyway, but showing the option would imply it's a valid choice.

## Code Duplication: The Right Call (For Now)

`create_storage.go` and `create_vault.go` share ~60% of their code. The state machine pattern, API submission, View rendering, and most Update logic are identical. The temptation to extract a generic `CreateClaimModel` is strong.

We didn't. Here's why:

1. **Two commands isn't a pattern.** Extracting a generic model for two consumers creates coupling without evidence it'll be needed for a third.
2. **The differences are in the middle.** Storage has tier/redundancy/versioning. Vault has SKU/retention. These aren't easily parameterized without making the generic model more complex than the specific ones.
3. **Each wizard can evolve independently.** Future storage features (blob access tiers, lifecycle policies) shouldn't be constrained by a shared abstraction designed for the vault flow.

The rule: **extract when you have three consumers, not two.** If `rdp scaffold create` follows the same pattern, that's the signal to extract shared form infrastructure.

## Key Takeaways

**1. Bubbletea's Elm Architecture maps well to form wizards.** One state per field, validate on transition, async commands for API calls. The pattern is predictable once you internalize it.

**2. Text inputs and selection lists need separate input routing.** Don't let `textinput.Model` consume keypresses during selection states. Check state before dispatching to the text input.

**3. Validate early, validate often, validate independently.** CLI validation for fast feedback. API validation for correctness. Gatekeeper for enforcement. Each layer operates independently and catches different failure modes.

**4. Graceful degradation > hard requirements.** Git auto-detection is nice. Manual entry is the fallback. The command always works.

**5. Duplicate two, extract on three.** Two similar implementations aren't enough to justify a generic abstraction. Wait for the third consumer to reveal the real pattern.

---

## References

- **Cobra commands:** [cli/cmd/infra_create.go](../../cli/cmd/infra_create.go)
- **Storage TUI:** [cli/internal/tui/create_storage.go](../../cli/internal/tui/create_storage.go)
- **Vault TUI:** [cli/internal/tui/create_vault.go](../../cli/internal/tui/create_vault.go)
- **Shared components:** [cli/internal/tui/shared.go](../../cli/internal/tui/shared.go)
- **Platform API endpoint:** [api/internal/infra/](../../api/internal/infra/)
- **Changelog:** [CHANGELOG.md](../../CHANGELOG.md)
