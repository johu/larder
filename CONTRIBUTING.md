# Contributing

Thank you for considering a contribution to **Larder**! Please follow the
guidelines below to keep the project history clean and the code consistent.

---

## Conventional Commits

All commits **must** follow the
[Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/)
specification:

```
<type>(<optional scope>): <description>

[optional body]

[optional footer(s)]
```

### Commit Types

| Type       | When to use                                           |
|------------|-------------------------------------------------------|
| `feat`     | A new feature (endpoint, field, search capability)    |
| `fix`      | A bug fix                                             |
| `docs`     | Documentation only changes                            |
| `style`    | Formatting only, no change in behaviour               |
| `refactor` | Code change that neither fixes a bug nor adds a feature |
| `perf`     | Performance improvement                               |
| `test`     | Adding or correcting tests                            |
| `build`    | Go modules, Dockerfile, or build tooling changes      |
| `ci`       | CI/CD pipeline changes                                |
| `chore`    | Maintenance that does not touch source or tests       |
| `revert`   | Reverts a previous commit                             |

Use a scope when it helps, for example `feat(api):`, `fix(store):` or
`build(docker):`.

Mark breaking changes (API or database schema) with `!` after the type or
scope, and explain them in a `BREAKING CHANGE:` footer.

### Commit Message Format

- **Subject line**: Keep within **50 characters** when possible, written in
  the imperative mood and without a trailing period.
- **Body**: Wrap lines at **72 characters**.
- **Body style**: Write as continuous text whenever possible. Use bullet
  points only when they materially improve clarity. Explain *why*, not just
  *what*.

### Examples

```
feat(api): add recipe search by ingredient

Allow filtering recipes by one or more ingredient names so that
"what can I cook with what I have" queries work without scanning
free text.
```

```
fix(store): keep ingredient order on update

Ingredients were re-inserted without their position, which shuffled
them after every edit. The position is now written explicitly.
```

---

## Branch Naming

| Pattern                        | Purpose                          |
|--------------------------------|----------------------------------|
| `feat/<short-description>`     | New features                     |
| `fix/<short-description>`      | Bug fixes                        |
| `docs/<short-description>`     | Documentation updates            |
| `refactor/<short-description>` | Restructuring without behaviour change |
| `test/<short-description>`     | Test additions or corrections    |
| `ci/<short-description>`       | CI/CD pipeline changes           |
| `chore/<short-description>`    | Maintenance/tooling              |
| `perf/<short-description>`     | Performance improvements         |
| `build/<short-description>`    | Go modules, Docker, or build changes |

---

## Editing Rules

Follow the [`.editorconfig`](.editorconfig) for every edit:

- **Character encoding**: UTF-8
- **Line endings**: LF (Unix-style)
- **Indentation**: Spaces, except tabs in Go, `go.mod`, `go.sum`, Makefiles
  and Caddyfiles
- **Indent size**: 2 spaces for most files, 4 for Go, shell scripts and
  Dockerfiles
- **Final newline**: Ensure files end with a newline
- **Trailing whitespace**: Trim, except in Markdown where it may be
  intentional (hard line breaks)
- **Markdown**: Wrap at 80 characters

Format Go code with `gofmt` before committing.

---

## Workflow

1. Branch from `main`.
2. Make your changes following the commit conventions above.
3. Make sure the code builds, is formatted (`gofmt`), and tests pass
   (`go test ./...`).
4. Open a pull request targeting `main`.
5. Ensure the title itself follows the Conventional Commits format.
6. Keep each request focused on one topic; split unrelated changes.
