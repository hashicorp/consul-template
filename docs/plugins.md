# Plugins

## Authoring Plugins

For some use cases, it may be necessary to write a plugin that offloads work to
another system. This is especially useful for things that may not fit in the
"standard library" of Consul Template, but still need to be shared across
multiple instances.

Consul Template plugins must have the following API:

```shell
$ NAME [INPUT...]
```

- `NAME` - the name of the plugin - this is also the name of the binary, either
  a full path or just the program name.  It will be executed in a shell with the
  inherited `PATH` so e.g. the plugin `cat` will run the first executable `cat`
  that is found on the `PATH`.

- `INPUT` - input from the template. There will be one INPUT for every argument passed
  to the `plugin` function. If the arguments contain whitespace, that whitespace
  will be passed as if the argument were quoted by the shell.

### Important Notes

- Plugins execute user-provided scripts and pass in potentially sensitive data
  from Consul or Vault. Nothing is validated or protected by Consul Template,
  so all necessary precautions and considerations should be made by template
  authors

- Plugin output must be returned as a string on stdout. Only stdout will be
  parsed for output. Be sure to log all errors, debugging messages onto stderr
  to avoid errors when Consul Template returns the value. Note that output to
  stderr will only be output if the plugin returns a non-zero exit code.

- Always `exit 0` or Consul Template will assume the plugin failed to execute

- Ensure the empty input case is handled correctly (see [Multi-phase execution](#multi-phase-execution))

- Data piped into the plugin is appended after any parameters given explicitly (eg `{{ "sample-data" | plugin "my-plugin" "some-parameter"}}` will call `my-plugin some-parameter sample-data`)

Here is a sample plugin in a few different languages that removes any JSON keys
that start with an underscore and returns the JSON string:

```ruby
#! /usr/bin/env ruby
require "json"

if ARGV.empty?
  puts JSON.fast_generate({})
  Kernel.exit(0)
end

hash = JSON.parse(ARGV.first)
hash.reject! { |k, _| k.start_with?("_")  }
puts JSON.fast_generate(hash)
Kernel.exit(0)
```

```go
func main() {
  arg := []byte(os.Args[1])

  var parsed map[string]interface{}
  if err := json.Unmarshal(arg, &parsed); err != nil {
    fmt.Fprintln(os.Stderr, fmt.Sprintf("err: %s", err))
    os.Exit(1)
  }

  for k, _ := range parsed {
    if string(k[0]) == "_" {
      delete(parsed, k)
    }
  }

  result, err := json.Marshal(parsed)
  if err != nil {
    fmt.Fprintln(os.Stderr, fmt.Sprintf("err: %s", err))
    os.Exit(1)
  }

  fmt.Fprintln(os.Stdout, fmt.Sprintf("%s", result))
  os.Exit(0)
}
```
