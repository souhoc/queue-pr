# queue-pr
WIP
Everything can change.

## Problem
A PR may stay open for too long.

## How to fix it?

## Install
go install github.com/souhoc/queue-pr@latest

## Usage
```sh
queue-pr -org <org> -token <token> 
```

## Example
```sh
# On mac. With `security`
security add-generic-password -a john@ecorp.com -s gh_api -w ghp_************************************

queue-pr -org ecorp -token $(security find-generic-password -a john@ecorp.com -s gh_api -w)
```
 

# TODO
## Config 
Having a templating system
i.e.:
```
Flags:
LABELS: gngengngn
NAME: Name of the PR
AUTHOR: Creator of the PR
Output: %LABELS %NAME %AUTHOR
```

## Features
- [ ] Filter on min duration
- [ ] Exclude or include repos
