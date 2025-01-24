# Installer

This project is the Installer for the Krateo platform. It sets up and manages the necessary components for the platform.

## Environment Variables

The following environment variables can be configured for the Installer:

- `DEBUG`: Run with debug logging. Default is `false`.
- `NAMESPACE`: Watch resources only in this namespace. Default is an empty string.
- `SYNC`: Controller manager sync period. Default is `1h`.
- `POLL_INTERVAL`: Poll interval controls how often an individual resource should be checked for drift. Default is `5m`.
- `MAX_RECONCILE_RATE`: The global maximum rate per second at which resources may be checked for drift from the desired state. Default is `3`.
- `LEADER_ELECTION`: Use leader election for the controller manager. Default is `false`.
- `MAX_ERROR_RETRY_INTERVAL`: The maximum interval between retries when an error occurs. This should be less than the half of the poll interval. Defaults is (`POLL_INTERVAL`/2)
- `MIN_ERROR_RETRY_INTERVAL`: The minimum interval between retries when an error occurs. Default is `1s`.

The following environment variables can be configured for the Helm client:

- `MAX_HELM_HISTORY`: The maximum number of helm releases to keep in history. Defaults is `10`