# EtherCAT Configurator

A lightweight, CLI-first EtherCAT SubDevice configuration generator and verifier.

The goal is to make EtherCAT SubDevice configuration reproducible, reviewable, and friendly to CI and agent workflows. A single declarative device file should become the source of truth for ESI XML, SII EEPROM binaries, firmware headers, and layout reports.

## Why This Project

EtherCAT configuration often crosses several artifacts at once: Object Dictionary entries, PDO mapping, Sync Manager sizes, ESI XML, EEPROM content, firmware offsets, and master-side expectations. In many small-device workflows those details are hidden in GUI projects, generated code, or handwritten offset tables.

EasyCAT-style generators can also hide fixed IN/OUT process data memory addresses in generated EEPROM and XML artifacts. Those defaults work for small examples, but longer process data layouts can outgrow the default buffer region and produce unsafe offsets unless the addresses are checked and made configurable.

This project aims to make those assets explicit:

- Device layout is declared in a human-reviewable file.
- IN/OUT process data memory addresses are configurable and validated against layout size.
- PDO bit and byte offsets are calculated, reported, and generated into firmware headers.
- ESI XML and SII EEPROM binaries are generated from the same intermediate model.
- CI can validate drift before hardware testing.
- AI agents and scripts can use deterministic commands and structured diagnostics.

## How It Fits

This project is not intended to replace existing EtherCAT runtime stacks or engineering tools. It is the configuration layer around them.

| Tool | Best At | Relationship |
| --- | --- | --- |
| SOES | Embedded EtherCAT SubDevice runtime stack | Firmware-side integration target and behavior reference |
| SOEM | Lightweight EtherCAT master library | Future verification backend and process data comparison reference |
| PySOEM | Python scripting around SOEM | Useful prototype/reference for validation workflows |
| IgH EtherCAT Master | Mature Linux master and diagnostics | Linux validation reference and CLI inspiration |
| ET9000 / TwinCAT / vendor tools | Compatibility, import, commissioning, and production engineering | Generated artifacts should remain importable, but daily generation should not depend on GUI-only workflows |

## CLI

```bash
ethercat-configurator --help
ethercat-configurator validate device.yaml
ethercat-configurator validate --help
ethercat-configurator report device.yaml -o report.md
ethercat-configurator gen esi device.yaml -o device.xml
ethercat-configurator gen esi --help
ethercat-configurator gen sii device.yaml -o eeprom.bin
ethercat-configurator gen header device.yaml -o ethercat_device.h
ethercat-configurator inspect esi device.xml
ethercat-configurator inspect sii eeprom.bin
ethercat-configurator diff expected.yaml actual.xml
```

Generated artifacts are deterministic and come from the same canonical model:

- `gen esi` writes a minimal ESI XML file with identity, Object Dictionary, PDO mappings, process data addresses, and Sync Manager metadata.
- `gen sii` writes a stable EEPROM binary container with identity, strings, Object Dictionary, PDO mappings, process data addresses, and checksum validation.
- `gen header` writes C/C++ constants, process data buffers, bit masks, byte offsets, and static assertions for firmware.
- `inspect esi` and `inspect sii` print normalized summaries; pass `--json` for machine-readable output.

## Current Status

The CLI foundation, YAML model loading, PDO layout, process data address validation, reports, ESI XML generation, SII EEPROM generation, and firmware header generation are implemented. See [PLAN.md](PLAN.md) for the roadmap and issue index. Detailed implementation scope and acceptance criteria live in GitHub Issues.
