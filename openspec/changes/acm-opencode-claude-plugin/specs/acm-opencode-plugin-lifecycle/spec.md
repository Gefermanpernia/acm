# acm-opencode-plugin-lifecycle Specification

## Purpose

Manage opt-in installation, migration, and rollback of the bundled experimental ACM OpenCode Claude plugin.

## Requirements

### Requirement: Bundled Experimental Opt-In

The ACM distribution MUST bundle the plugin but MUST keep it experimental and disabled until explicit user opt-in. It MUST NOT automatically replace a stable or upstream plugin. The installer and lifecycle command MUST resolve the bundled entry point from `ACM_SHARE_DIR/opencode/index.js` when `ACM_SHARE_DIR` is set, then use `$HOME/.local/share/acm/opencode/index.js` only when the override is unset. If the override is set but its entry point is missing or is not a regular file, the lifecycle command MUST fail closed without falling back to the default path or changing the OpenCode configuration.

#### Scenario: Fresh ACM installation

- GIVEN ACM is installed without an opt-in request
- WHEN installation completes
- THEN the bundled plugin SHALL remain disabled and upstream configuration unchanged.

#### Scenario: User explicitly enables the experiment

- GIVEN a compatible Linux installation and explicit opt-in
- WHEN guided installation completes successfully
- THEN ACM MAY enable the experimental plugin.

#### Scenario: Custom share installation remains enable-able

- GIVEN the installer staged the plugin under an explicit `ACM_SHARE_DIR`
- WHEN the user explicitly enables the experiment with the same environment
- THEN the lifecycle command SHALL configure and load that staged entry point.

#### Scenario: Configured custom share entry point is missing

- GIVEN `ACM_SHARE_DIR` is set but its plugin entry point is missing or unsafe
- WHEN the user explicitly enables the experiment
- THEN the lifecycle command MUST fail without using the default share path or changing the OpenCode configuration.

### Requirement: Guided Exclusive Migration

Migration MUST require explicit confirmation, create a restorable backup of the OpenCode configuration, and enforce mutual exclusivity between the ACM plugin and the upstream plugin.

#### Scenario: Confirmed migration from upstream plugin

- GIVEN the upstream plugin is configured and the user confirms migration
- WHEN migration is performed
- THEN ACM SHALL back up the configuration and enable only the ACM plugin.

#### Scenario: Plugin conflict is detected

- GIVEN both ACM and upstream plugin entries are present
- WHEN installation or migration is requested
- THEN the lifecycle command MUST stop with a conflict until guided resolution is confirmed.

### Requirement: Configuration-Only Rollback

Rollback MUST restore the backed-up upstream OpenCode configuration, disable the ACM plugin, and MUST NOT migrate, rewrite, or expose ACM account state.

#### Scenario: Rollback after experimental use

- GIVEN a valid migration backup exists
- WHEN the user requests rollback
- THEN the upstream configuration SHALL be restored and the ACM plugin disabled.

#### Scenario: Backup is missing or invalid

- GIVEN no valid restorable backup exists
- WHEN rollback is requested
- THEN the command MUST fail closed without changing OpenCode or ACM state.
