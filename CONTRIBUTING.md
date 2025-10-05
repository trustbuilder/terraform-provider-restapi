# 🤝 Contributing

## 🚀 Getting Started

```
git clone https://github.com/trustbuilder/terraform-provider-trustbuilder.git
cd terraform-provider-trustbuilder
```

## 🛠️ Requirements
- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.23
- [golangci-lint](https://github.com/golangci/golangci-lint?tab=readme-ov-file#install-golangci-lint)
- [pre-commit](https://pre-commit.com/#installation)
- [make](https://www.gnu.org/software/make/manual/make.html)


## 🐧 Development setup for Linux
1. Define the environment variable `GOBIN` if you want the another default Go binary location elsewhere that `~/go/bin`
2. Define the Terraform provider location in development in `~/.terraformrc`:
    ```
    provider_installation {
      dev_overrides {
          "trustbuilder" = "/home/<USERNAME>/go/bin"
      }

      # For all other providers, install them directly from their origin provider
      # registries as normal. If you omit this, Terraform will _only_ use
      # the dev_overrides block, and so no other providers will be available.
      direct {}
    }
    ```
3. pre-commit
    ```
    pre-commit install
    pre-commit run -all-files
    ```

## Building The Provider
1. Clone the repository: `git clone https://github.com/trustbuilder/terraform-provider-trustbuilder.git`
2. Enter the repository directory: `cd terraform-provider-trustbuilder`
3. Build the provider : `make build`


## 🐛 Debugging
* For [VSCode](https://registry.terraform.io/providers/DigitecGalaxus/dg-servicebus/latest/docs/guides/howto-debugprovider)
* When you run the tests, if you want to see the **all the logs**, you have to set `TF_LOG="DEBUG"`.


## ✅ Execute the tests
* unit tests:
  ```bash
  make test
  ```
* acceptance tests:
  ```bash
  make testacc
  ```


## 📦 Adding Dependencies

This provider uses [Go modules](https://github.com/golang/go/wiki/Modules).
Please see the Go documentation for the most up to date information about using Go modules.

To add a new dependency `github.com/author/dependency` to your Terraform provider:

```shell
go get github.com/author/dependency
go mod tidy
```

Then commit the changes to `go.mod` and `go.sum`.


## 📄 Generate the provider documentation

```bash
make generate
```
