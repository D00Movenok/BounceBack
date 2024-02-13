# BounceBack

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/D00Movenok/BounceBack)](https://goreportcard.com/report/github.com/D00Movenok/BounceBack)
[![Tests](https://github.com/D00Movenok/BounceBack/actions/workflows/tests.yml/badge.svg)](https://github.com/D00Movenok/BounceBack/actions/workflows/tests.yml)
[![CodeQL](https://github.com/D00Movenok/BounceBack/actions/workflows/codeql.yml/badge.svg)](https://github.com/D00Movenok/BounceBack/actions/workflows/codeql.yml)
[![Docs](https://img.shields.io/badge/docs-wiki-blue?logo=GitBook)](https://github.com/D00Movenok/BounceBack/wiki)

↕️🤫 Stealth redirector for your red team operation security.

![Atchitecture](/assets/architecture.png)

## Overview

BounceBack is a powerful, highly customizable and configurable reverse proxy with WAF functionality for hiding your C2/phishing/etc infrastructure from blue teams, sandboxes, scanners, etc. It uses real-time traffic analysis through various filters and their combinations to hide your tools from illegitimate visitors.

The tool is distributed with preconfigured lists of blocked words, blocked and allowed IP addresses.

For more information on tool usage, you may visit [project's wiki](https://github.com/D00Movenok/BounceBack/wiki).

## Features

* Highly configurable and customizable filters pipeline with boolean-based concatenation of rules will be able to hide your infrastructure from the most keen blue eyes.
* Easily extendable project structure, everyone can add rules for their own C2.
* Integrated and curated massive blacklist of IPv4 pools and ranges known to be associated with IT Security vendors combined with IP filter to disallow them to use/attack your infrastructure.
* Malleable C2 Profile parser is able to validate inbound HTTP(s) traffic against the Malleable's config and reject invalidated packets.
* Out of the box domain fronting support allows you to hide your infrastructure a little bit more.
* Ability to check the IPv4 address of request against IP Geolocation/reverse lookup data and compare it to specified regular expressions to exclude out peers connecting outside allowed companies, nations, cities, domains, etc.
* All incoming requests may be allowed/disallowed for any time period, so you may configure work time filters.
* Support for multiple proxies with different filter pipelines at one BounceBack instance.
* Verbose logging mechanism allows you to keep track of all incoming requests and events for analyzing blue team behaviour and debug issues.

## Rules

The main idea of rules is how BounceBack matches traffic. The tool currently supports the following rule types:

* Boolean-based (and, or, not) rules combinations
* IP and subnet analysis
* IP geolocation fields inspection
* Reverse lookup domain probe
* Raw packet regexp matching
* Malleable C2 profiles traffic validation
* Work (or not) hours rule

Custom rules may be easily added, just register your [RuleBaseCreator](/internal/rules/default.go#L9) or [RuleWrapperCreator](/internal/rules/default.go#L3). See already created [RuleBaseCreators](/internal/rules/base_common.go) and [RuleWrapperCreators](/internal/rules/wrappers.go)

Rules configuration page may be found [here](https://github.com/D00Movenok/BounceBack/wiki/1.-Rules).

## Proxies

The proxies section is used to configure where to listen and proxy traffic, which protocol to use and how to chain rules together for traffic filtering. At the moment, BounceBack supports the following protocols:

* HTTP(s) for your web infrastructure
* DNS for your DNS tunnels
* Raw TCP (with or without tls) and UDP for custom protocols

Custom protocols may be easily added, just register your new type [in manager](/internal/proxy/manager.go). Example proxy realizations may be found [here](/internal/proxy).

Proxies configuration page may be found [here](https://github.com/D00Movenok/BounceBack/wiki/2.-Proxies).

## Installation

Just download latest release from [release page](https://github.com/D00Movenok/BounceBack/releases), unzip it, edit config file and go on.

If you want to build it from source, clone it (don't forget about [GitLFS](https://git-lfs.com/)), [install goreleaser](https://goreleaser.com/install/) and run:

```bash
goreleaser release --clean --snapshot
```

## Usage

1. **(Optionally)** Update `banned_ips.txt` list:

    ```bash
    bash scripts/collect_banned_ips.sh > data/banned_ips.txt
    ```

2. Modify `config.yml` for your needs. Configure [rules](https://github.com/D00Movenok/BounceBack/wiki/1.-Rules) to match traffic, [proxies](https://github.com/D00Movenok/BounceBack/wiki/2.-Proxies) to analyze traffic using rules and [globals](https://github.com/D00Movenok/BounceBack/wiki/3.-Globals) for deep rules configuration.

3. Run BounceBack:

    ```bash
    ./bounceback
    ```

    > Usage of BounceBack: \
    > -c, --config string   Path to the config file in YAML format (default "config.yml") \
    > -l, --log string      Path to the log file (default "bounceback.log") \
    > -v, --verbose count   Verbose logging (0 = info, 1 = debug, 2+ = trace)
