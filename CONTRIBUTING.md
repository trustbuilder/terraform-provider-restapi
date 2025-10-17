# 🤝 Contributing

# 🛠️ Requirements
- [asdf](https://asdf-vm.com/guide/getting-started.html#_1-install-asdf)

OR:

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.23
- [golangci-lint](https://github.com/golangci/golangci-lint?tab=readme-ov-file#install-golangci-lint)
- [pre-commit](https://pre-commit.com/#installation)

# 🚀 Getting Started

## 🐧 Development setup for Linux
1. Install requirements with asdf (if you chose this solution)
    ```
    asdf current
    # Install a version of terraform globally for the acceptance tests
    asdf set -u terraform latest
    asdf install
    ```
2. Define the environment variable `GOBIN` if you want the another default Go binary location elsewhere that `~/go/bin`
3. Use the provider development version
    You can plan and apply locally with the version in development of this provider with:
    ```bash
    make install
    export TF_CLI_CONFIG_FILE=./dev.tfrc
    ```
  It avoids to modify directly your `~/.terraformrc` file
4. pre-commit
    ```bash
    pre-commit install
    pre-commit run --all-files
    ```

## 🐛 Debugging
* For [VSCode](https://registry.terraform.io/providers/DigitecGalaxus/dg-servicebus/latest/docs/guides/howto-debugprovider)
* When you run the tests, if you want to see the **all the logs**, you have to set `TF_LOG="DEBUG"`.


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
