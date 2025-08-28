# 🤝 Contributing

# 🛠️ Requirements
- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.23
- [golint]

# 🚀 Getting Started

## 🐧 Development setup for Linux
1. Define the environment variable `GOBIN` if you want the another default Go binary location elsewhere that `~/go/bin`
2. Define the Terraform provider location in development in `~/.terraformrc`:
    ```
    provider_installation {
    dev_overrides {
        "restapi" = "/home/<USERNAME>/go/bin"
    }

    # For all other providers, install them directly from their origin provider
    # registries as normal. If you omit this, Terraform will _only_ use
    # the dev_overrides block, and so no other providers will be available.
    direct {}
    }
    ```

## 🐛 Debugging
* For [VSCode](https://registry.terraform.io/providers/DigitecGalaxus/dg-servicebus/latest/docs/guides/howto-debugprovider)


# ✅ Execute the tests
* unit tests:
  ```bash
  make test
  ```
* acceptance tests:
  ```bash
  make testacc
  ```

# 📄 Generate the provider documentation
```bash
make generate
```
