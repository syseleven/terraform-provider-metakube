#!/usr/bin/env bash

cat >~/.terraformrc <<EOS
provider_installation {
  dev_overrides {
    "syseleven/metakube" = "$(realpath $(dirname $0))"
  }
}
EOS

