# ingress-anubis

WIP ingress controller for [anubis].

## Disclaimer

This is NOT AT ALL production software and may never be. Likely bugs
that will exist are: lack of upwards flowing reconciliation, perfect
garbage control, and the like. State management is hard :smile:

## Goals

 - Wrap [ingress-nginx] with anubis in front of simple ingresses (one
   service).
 - Only require `ingressClassName` to be set to `anubis` to enable, and
   summarily disable.

## Development

We use `mise` to manage the versions of our tools in usage as well as
for task management. Check out the [mise] documentation to get started!

## License

GPL-3.0

[anubis]: https://github.com/TecharoHQ/anubis
[mise]: https://mise.jdx.dev
[ingress-nginx]: https://github.com/kubernetes/ingress-nginx
