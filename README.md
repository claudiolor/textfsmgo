# TextFSMGo

TextFSMGo is just another implementation of [Google TextFSM](https://github.com/google/textfsm), a template-based state machine for parsing semi-formatted text. To write a template file check the [original project documentation](https://github.com/google/textfsm/wiki/TextFSM).

TextFSMGo is compatible with the existing TextFSM templates, with the following caveats:

- Named match groups in values definition are currently not supported, so values cannot contain dictionaries;
- Perl syntax of regex is not supported.

---

:warning: I wrote TextFSMGo just for fun and to practice with Golang, so please take it as is.

---

## Build and install

To build and install TextFSMGo is it possible to leverage the Makefile recipe.

To build the project from the root directory of this repository run the command:

```shell
make build
```

it will build and store the executable in the `./dist` directory of the repo.

To build and install TextFSMGo launch the command:

```shell
make install
```



