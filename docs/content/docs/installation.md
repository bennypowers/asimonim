---
title: "Installation"
weight: 10
---

## npm

```bash
npm install -g @pwrs/asimonim
```

## Gentoo Linux

Enable the `bennypowers` overlay, then install:

```bash
eselect repository enable bennypowers
emaint sync -r bennypowers
emerge dev-util/asimonim
```

## From Source

```bash
go install bennypowers.dev/asimonim@latest
```
