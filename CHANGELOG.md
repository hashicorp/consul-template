Consul Template Changelog
=========================

v0.1.1
------
### Bug Fixes
- Fixed an issue where help output was displayed twice when specifying the `-h`
	flag
- Added support for specifyiny forward slashes (`/`) in service names
- Added support for specifying underscores (`_`) in service names
- Added support for specifying dots (`.`) in tag names

### Improvements
- Added support for Travis CI
- Fixed numerous typographical errors
- Added more documentation, including an FAQ in the README
- Do not return an error when a template has no dependencies. See GH-31 for more
	background and information
- Do not render templates if they have the same content
- Do not execute commands if the template on disk would not be changed

v0.1.0
------
- Initial release
