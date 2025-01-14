# with-secrets

This is a wrapper command to execute command with AWS Secrets Manager, and mask secrets from its output.

## Usage

```console
$ export WITH_SECRETS_ID=my-secret-id
$ with-secrets my-command

or you can run simply
$ WITH_SECRETS_ID=my-secret-id with-secrets my-command
```
