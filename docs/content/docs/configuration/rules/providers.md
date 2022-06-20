---
title: "Providers"
date: 2022-06-09T22:13:54+02:00
lastmod: 2022-06-09T22:13:54+02:00
draft: true
menu:
  docs:
    weight: 30
    parent: "Rules"
---

Providers define the sources to load the rule sets from. These make Heimdall's behavior dynamic. All providers, you want to enable for a Heimdall instance must be configured by within the `providers` section of Heimdall's `rules` configuration.

Supported providers, including the corresponding configuration options are described below

## Filesystem

The filesystem provider allows loading of rule sets from a file system. The configuration of this provider goes into the `file` property. This provider is handy for e.g. starting playing around with Heimdall, e.g. locally, or using Docker, as well as if your deployment strategy considers deploying a Heimdall instance as a Side-Car for each of your services. 

Following configuration options are supported:

| Name    | Type      | Mandatory | Description                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                             |
|---------|-----------|-----------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `src`   | *string*  | yes       | Can either be a single file, containing a rule set, or a directory with files, each containing a rule set.                                                                                                                                                                                                                                                                                                                                                                                                                                              |
| `watch` | *boolean* | no        | Whether the configured `src` should be watched for updates. Defaults to `false`. If the `src` has been configured to a single file, the provider will watch for changes in that file. Otherwise, if the `src` has been configured to a directory, the provider will watch for files appearing and disappearing in this directory, as well as for changes in each particular file in this directory. Recursive lookup is not supported. That is, if the configured directory contains further directories, these, as well as their contents are ignored. |

This provider doesn't need any additional configuration for a rule set. So the contents of files can be just a list of rules as described in [Rule Sets]({{< relref "rule_definition.md#rule-set" >}}).

**Example 1**

In this example, the filesystem provider is configured to load rule sets from the files residing in the  `/path/to/rules/dir` directory and watch for changes.

```yaml
providers:
  file:
    src: /path/to/rules/dir
    watch: true
```

**Example 2**

In this example, the filesystem provider is configured to load rule sets from the `/path/to/rules.yaml` file without watching it for changes.

```yaml
providers:
  file:
    src: /path/to/rules.yaml
```