Consul Template Changelog
=========================

## v0.7.1.dev (Unreleased)

FEATURES:

  * Add `.Size()` so the watcher can report its size (GH-206)
  * Add `byKey` template helper to group the results of a `tree` function by
    their containing directory (GH-207, GH-209)
  * Add `timestamp` template function for returning the current timestamp with
    the ability to add custom formatting (GH-225, GH-230)
  * Add `loop` template function for iteration (GH-238, GH-221)

IMPROVEMENTS:

  * Expose `LastIndex` and `ReceivedData` from the Watcher
  * Add unimplemented KV fields (GH-203)
  * Warn the user if there are a large number of dependencies (GH-205)
  * Extend documentation on how health service dependencies are downloaded from
    Consul (GH-212)
  * Allow empty configuration directories (GH-217)
  * Document caveats around using `parseJSON` during multi-evaluation
  * Print the final configuration as JSON in debug mode (GH-231)
  * Export certain environment variables when executing commands that are read
    by other Consul tooling or in your scripts (GH-232) - see the README for
    more information

BUG FIXES:

  * Properly filter services by their type (GH-210, GH-212)
  * Return an error if extra arguments are given on the command line (GH-227)
  * Do not overwrite given configuration with the default options (GH-228, GH-219)
  * Check for the correct conditions when using basic authentication (GH-220)

## v0.7.0 (February 19, 2015)

BREAKING CHANGES:

  * Remove `ssl` configuration option from templates - use an `ssl`
    configuration block with `enabled = true` instead
  * Remove `ssl_no_verify` configuration option from templates - use an `ssl`
    configuration block with `verify = false` instead
  * Restructure CLI `-ssl-no-verify` to `-ssl-verify` - to disable SSL
    certification validation on the command line, use `-ssl-verify=false`
  * Remove `auth` configuration option from templates - use an `auth`
    configuration block with `enabled = true` combined with `username = ...` and
    `password = ...` inside the block instead

FEATURES:

  * Add support for logging to syslog (GH-163)
  * Add `log_level` as a configuration file option
  * Add `-log-level` as a CLI option

IMPROVEMENTS:

  * Use a default retry interval of 5s (GH-190) - this value has been (and will
    remain) configurable since v0.5.0, but the default value has changed from 0
    to 5s
  * Use a service's reported address if given (GH-185, GH-186)
  * Add new `NodeAddress` field to health services to always include the node's
    address
  * Return errors up the watcher's error channel so other libraries can
    determine what to do with the error instead of swallowing it (GH-196)
  * Move SSL and authentication options into their own configuration blocks in
    the HCL
  * Add new `watch.WaitVar` for parsing Wait structs via Go's flag parsing
    library.
  * Extract logging components into their own library for sharing (GH-199)

BUG FIXES:

  * Return errors instead of nil in catalog nodes and key prefix dependencies
    (GH-192)
  * Allow Consul Template to exit when running in `once` mode and templates have
    not changed (GH-188)
  * Raise an error when specifying a non-existent option in the configuration
    file (GH-197)
  * Use an RWLock when accessing information in the Brain to improve performance
  * Improve debugging output and consistency
  * Remove unused Brain functions
  * Remove unused documentation items
  * Use the correct default values for `-ssl` and `-retry` on the CLI

## v0.6.5 (February 5, 2015)

FEATURES:

  * Add `-max-stale` to specify Consul Template may talk to non-leader Consul
    nodes if they are less than the maximum stale value (GH-183)

BUG FIXES:

  * Fix a concurrency bug in the Brain (GH-180)
  * Add a better queue-draining mechanism for templates that have a large number
    of dependencies (GH-184)

## v0.6.1 (February 2, 2015)

IMPROVEMENTS:

  * Allow watcher to use buffered channels so we do not block when multiple
    dependencies return data (GH-176)
  * Buffer results from the watcher to reduce the number of CPU cycles (GH-168
    and GH-178)

BUG FIXES:

  * Handle the case where reloading via SIGHUP would cause an error (GH-175 and
    GH-177)
  * Return errors to the template when parsing a key fails (GH-170)
  * Expand the list of possible values for keys to non-ASCII fields (the `@` is
    still a restricted character because it denotes the datacenter) (GH-170)
  * Diff missing dependencies during the template render to avoid creating
    extra watchers (GH-169)
  * Improve debugging output (GH-169)

## v0.6.0 (January 20, 2015)

FEATURES:

  * Implement n-pass evaluation (GH-64) - templates are now evaluated N+1 times
    to properly accumulate dependencies and build the graph properly

BREAKING CHANGES:

  * Remove `storeKeyPrefix` template function - it has been replaced with `ls`
    and/or `tree` and was deprecated in 0.2.0
  * Remove `Key()` from dependency interface

IMPROVEMENTS:

  * Switch to using `hashicorp/consul/api` instead of `armon/consul-api`
  * Add support for communicating with Consul via HTTPS/SSL (GH-143)
  * Add support for communicating with Consul via BasicAuth (GH-147)
  * Quiesce on a per-template basis

BUG FIXES:

  * Reduce memory footprint when running with a large number of templates by
    using a single context instead of separate template contexts for each
    template
  * Improve test coverage
  * Improve debugging output
  * Correct tag deep copy that could result in 2N-1 tags (GH-155)
  * Return an empty slice when parsing an empty JSON file
  * Update README documentation

## v0.5.1 (December 25, 2014)

BUG FIXES:

  * Parse Retry values in the config (GH-136)
  * Remove `util` package as it is a code smell and separate `Watcher` and
    `Dependency` structs and functions into their own packages for re-use
    (GH-137)

## v0.5.0 (December 19, 2014)

FEATURES:

  * Reload configuration on `SIGHUP`
  * Add `services` template function for listing all services and associated
    tags in the Consul catalog (GH-77)

BUG FIXES:

  * Do not execute the same command more than once in one run (GH-112)
  * Do not exit when Consul is unavailable (GH-103)
  * Accept configuration files as a valid option to `-config` (GH-126)
  * Accept Windows drive letters in template paths (GH-78)
  * Deep copy and sort data returned from Consul API (specifically tags)
  * Run commands even if not all templates have received data (GH-119)

IMPROVEMENTS:

  * Add support for more complex service health filtering (GH-116)
  * Add support for specifying a `-retry` interval for Consul timeouts and
    connection errors (GH-22)
  * Use official HashiCorp multierror package for errors
  * Gracefully stop watchers on interrupt
  * Add support for Go 1.4
  * Improve test coverage around retrying failures

## v0.4.0 (December 10, 2014)

FEATURES:

  * Add `env` template function for reading an environment variable in the
    current process into the template
  * Add `regexReplaceAll` template function

BUG FIXES:

  * Fix documentation examples
  * Fix `golint` and `go vet` errors
  * Fix a panic when Consul returned empty query metadata
  * Allow colons in key prefixes (`ls` and `tree` receive this by proxy)
  * Allow `parseJSON` to handle top-level JSON objects
  * Filter empty keys in `tree` and `ls` (folder nodes)

IMPROVEMENTS:

  * Merge multiple configuration template definitions when a configuration
    directory is specified

## v0.3.1 (November 24, 2014)

BUG FIXES:

  * Allow colons in key names (GH-67)
  * Fix a documentation bug in the README in the Varnish example (GH-82)
  * Attempt to render templates before starting the watcher - this fixes an
    issue where a template that declared no Consul dependencies would never be
    rendered (GH-85)
  * Update inline Go documentation for better clarity

IMPROVEMENTS:

  * Fix all issues raised by `go vet`
  * Update packaging script to fix ZSHisms and use awk for clarity

## v0.3.0 (November 13, 2014)

FEATURES:

  * Added a `Contains` method to `Service.Tags`
  * Added support for specifying a configuration directory in `-config`, in
    addition to a file
  * Added support for querying all nodes in Consul's catalog with the `nodes`
    template function

BUG FIXES:

  * Update README documentation to clarify that `service` dependencies default
    to the current datacenter if one is not explicitly given
  * Ignore empty keys that are returned from an `ls` call (GH-54)
  * When writing a file atomicly, ensure the drive is the same (GH-58)
  * Run all commands before exiting - previously if a single command failed in
    a multi-template environment, the other commands would not execute, but
    Consul Template would return

IMPROVEMENTS:

  * Added support for querying all `service` nodes by passing an additional
    parameter to `service`

## v0.2.0 (November 4, 2014)

FEATURES:

  * Added helper for decoding a result as JSON using the `parseJSON` pipe
    function
  * Added support for reading and watching changes from a file using the `file`
    template function
  * Added helper for sorting service entires by a particular tag
  * Added helper function `toLower()` for converting a string to lowercase
  * Added helper function `toTitle()` for converting a string to titlecase
  * Added helper function `toUpper()` for converting a string to uppercase
  * Added helper function `replaceAll()` for replacing occurrences of a
    substring with a new string
  * Added `tree` function for returning all key prefixes recursively
  * Added `ls` function for returning all keys in the top-level prefix (but not
    deeply nested ones)

BUG FIXES:

  * Remove prefixes from paths when querying a key prefix

IMPROVEMENTS:

  * Moved shareable functions into a util module so other libraries can benefit
  * Make Path a public field on Template
  * Added more examples and documentation to the README

DEPRECATIONS:

  * `keyPrefix` is deprecated in favor or `tree` and `ls` and will be removed in
  the next major release


## v0.1.1 (October 28, 2014)

BUG FIXES:

  * Fixed an issue where help output was displayed twice when specifying the
    `-h` flag
  * Added support for specifyiny forward slashes (`/`) in service names
  * Added support for specifying underscores (`_`) in service names
  * Added support for specifying dots (`.`) in tag names

IMPROVEMENTS:

  * Added support for Travis CI
  * Fixed numerous typographical errors
  * Added more documentation, including an FAQ in the README
  * Do not return an error when a template has no dependencies. See GH-31 for
    more background and information
  * Do not render templates if they have the same content
  * Do not execute commands if the template on disk would not be changed

## v0.1.0 (October 21, 2014)

  * Initial release
