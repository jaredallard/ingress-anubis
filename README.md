# ingress-anubis

Kubernetes ingress controller for [anubis].

## Disclaimer

This is NOT AT ALL production software and may never be. See
[limitations](#limitations) for issues that are present and may never be
fixed!

## Goals

- Wrap [ingress-nginx] with anubis in front of simple ingresses (one
  service).
- Only require `ingressClassName` to be set to `anubis` to enable, and
  summarily disable.

## Limitations

In line with the above goals, the following limitations are currently
present:

- An ingress with more than one target will only point to the first
  found target. This is because anubis only supports one target and this
  controller only manages one instance of anubis per ingress, currently.
- Resources created by the controller are not reconciled if outside
  changes occur unless the source ingress is updated, triggering the
  reconciliation loop.

## Installing

Your cluster should already have the following installed:

- [ingress-nginx]

```bash
helm install --create-namespace --namespace ingress-anubis \
  ingress-anubis oci://ghcr.io/jaredallard/helm-charts/ingress-anubis
```

## Configuration

For available configuration options, see the `config` key in
[`values.yaml`](./deploy/charts/ingress-anubis/values.yaml), or in
[`config.go`](./internal/config/config.go) until further documentation is
written.

### Ingress Configuration

Anubis can be configured per-ingress through the following annotations,
on the ingress:

- ingress-anubis.jaredallard.github.com/serve-robots-txt (bool)
- ingress-anubis.jaredallard.github.com/og-passthrough (bool)
- ingress-anubis.jaredallard.github.com/difficulty (int)
- ingress-anubis.jaredallard.github.com/ingress-class (string)
  - Set the ingressClassName value for the wrapped ingress. The default
    is `nginx`. Note that `nginx` is the only officially supported
    setup right now.

See [anubis environment variable
documentation](https://anubis.techaro.lol/docs/admin/installation) for
more information on these values and what they do.

### Multiple Instances

Multiple instances of ingress-anubis can be ran under **different**
ingress class names, by setting `INGRESS_CLASS_NAME` (see
[Configuration](#configuration) for how to set this).

However, note that they must be ran in different namespaces as well.

## Usage

Once [installed](#installing), simply set `ingressClassName` to `anubis`
and watch as your site is now behind it :tada:!

To configure various settings, see [configuration](#configuration).

## Development

We use `mise` to manage the versions of our tools in usage as well as
for task management. Check out the [mise] documentation to get started!

## License

GPL-3.0

[anubis]: https://github.com/TecharoHQ/anubis
[mise]: https://mise.jdx.dev
[ingress-nginx]: https://github.com/kubernetes/ingress-nginx
