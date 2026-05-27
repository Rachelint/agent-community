# Agent instructions for agent-community

This file documents repo-level guidance for agents that are started from
this repository to discuss or implement work in other projects through the
agent-community platform.

## Default operating model

Assume the chat agent may be started from **this repository** by default.
The discussion may still be about some other project managed by the
platform.

Do not assume the current working directory is the target project repo.
Resolve the target project first.

## Project resolution

A common user prompt is effectively:

- "Let's discuss project X"
- "Let's discuss how to implement requirement Y in project X"

When the user names a project, the agent must:

1. Use the `platform-ops` skill first.
2. Pull the current project list from the platform.
3. Match the user-mentioned project name to a platform project.
4. Find that project's current local repository path.
5. Base all subsequent code reading, exploration, and implementation on that
   resolved project path.

If multiple projects could match the same name, ask the user to disambiguate
before exploring code.

## Branch-specific exploration

Discussions are often about a **specific branch**.

This rule is critical:

- Never switch branches directly inside the target repository checkout.
- Never run `git checkout`, `git switch`, or equivalent branch-changing
  commands in the user's working checkout of the target repo.

The user may be actively developing or inspecting that repository, and
changing the branch in-place can disrupt their work.

### Required approach

For any branch-specific exploration, implementation, or validation, the agent
must use a **git worktree**.

That means:

1. Keep the user's target repository checkout untouched.
2. Create or use a separate worktree for the requested branch.
3. Perform branch-specific reads, edits, builds, and tests inside that
   worktree only.

If the branch name is not known, ask the user which branch should be used.

## Practical workflow

When the user says, "Let's discuss project X and how to implement Y":

1. Use `platform-ops` to inspect platform projects.
2. Resolve project name -> project id -> repository path.
3. Determine whether the discussion depends on a specific branch.
4. If yes, create/use a worktree for that branch.
5. Explore the code in the resolved repo/worktree.
6. Keep any branch-changing operations away from the user's main checkout.

## Safety priority

Protecting the user's in-progress work in the target repository is more
important than convenience.

If there is any uncertainty about:

- which project the user means
- which branch should be inspected
- whether a branch-specific worktree already exists

ask before proceeding.