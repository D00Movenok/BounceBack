# BounceBack

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/D00Movenok/BounceBack)](https://goreportcard.com/report/github.com/D00Movenok/BounceBack)
[![Tests](https://github.com/D00Movenok/BounceBack/actions/workflows/tests.yml/badge.svg)](https://github.com/D00Movenok/BounceBack/actions/workflows/tests.yml)
[![CodeQL](https://github.com/D00Movenok/BounceBack/actions/workflows/codeql.yml/badge.svg)](https://github.com/D00Movenok/BounceBack/actions/workflows/codeql.yml)

↕️🤫 Stealth redirector for your red team operation security.

![Atchitecture](/assets/architecture.png)

## Overview

BounceBack is a powerful, highly customizable and configurable reverse proxy with WAF functionality for hiding your C2/phishing/etc infrastructure from blue teams, sandboxes, scanners, etc. It uses real-time traffic analysis through various filters and their combinations to hide your tools from illegitimate visitors.

The tool is distributed with preconfigured lists of blocked words, blocked and allowed IP addresses.

## Features

* Highly configurable and customizable filters pipeline with boolean-based concatenation of filters will be able to hide your infrastructure from the most keen blue eyes.
* Easily extendable project structure, everyone can add filters for their own C2.
* Integrated and curated massive blacklist of IPv4 pools and ranges known to be associated with IT Security vendors combined with IP filter to disallow them to use/attack your infrastructure.
* Malleable C2 Profile parser is able to validate inbound HTTP(s) traffic against the Malleable's config and reject invalidated packets.
* Out of the box domain fronting support allows you to hide your infrastructure a little bit more.
* Ability to check the IPv4 address of request against IP Geolocation/reverse lookup data and compare it to specified regular expressions to exclude out peers connecting outside allowed companies, nations, cities, domains, etc.
* All incoming requests may be allowed/disallowed for any time period, so you may configure work time filters.
* Support for multiple proxies with different filter pipelines at one BounceBack instance.
* Verbose logging mechanism allows you to keep track of all incoming requests and events for analyzing blue team behaviour and debug issues.

## Filters

BounceBack currently supports the following filters:

* Boolean-based (and, or, not) filters combinations
* IP and subnet analysis
* IP geolocation fields inspection
* Reverse lookup domain probe
* Raw packet regexp matching
* Malleable C2 profiles traffic validation
* Work (or not) hours filter

Custom filters may be easily added, just register your [FilterBaseCreator](/internal/filters/default.go#L9) or [FilterWrapperCreator](/internal/filters/default.go#L3). See already created [FilterBaseCreators](/internal/filters/base_common.go) and [FilterWrapperCreators](/internal/filters/wrappers.go)

Filters configuration page may be found [here](https://github.com/D00Movenok/BounceBack/wiki/1.-Filters).

## Proxies

At the moment, BounceBack supports the following protocols:

* HTTP(s) for your web infrastructure
* DNS for your DNS tunnels
* Raw TCP (with or without tls) and UDP for custom protocols

Custom protocols may be easily added, just register your new type [in manager](/internal/proxy/manager.go). Example proxy realizations may be found [here](/internal/proxy).

Proxies configuration page may be found [here](https://github.com/D00Movenok/BounceBack/wiki/2.-Proxies).

## Installation

Just download latest release from [release page](https://github.com/D00Movenok/BounceBack/releases), unzip it, edit config file and go on.

If you want to build it from source, [install goreleaser](https://goreleaser.com/install/) and run:

```bash
goreleaser release --clean --snapshot
```
