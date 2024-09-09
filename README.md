- Ignore symlinks / hardlinks
- Read home folder (Downloads, Documents, steam etc...) first

- Deleting (multiple selected) files
- Special case for /swapfile, suggest or run clear swap thing

[The command-line parser stops parsing after the first non-option](https://stackoverflow.com/a/25113485).\
This is valid:
`fssize --ignore-hidden-files .`

While this will not ignore hidden files:
`fssize . --ignore-hidden-files`

Tabs: Files Folders
