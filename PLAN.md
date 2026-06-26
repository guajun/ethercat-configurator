# EtherCAT Configurator Roadmap

这份文件是项目路线图和 issue 索引。具体任务的 single truth 在 GitHub Issues；这里只保留阶段顺序、设计边界和执行约束，避免 `PLAN.md` 与 issue 双写后漂移。

## 当前进度

- PR [#19](https://github.com/guajun/ethercat-configurator/pull/19) 正在将 artifact generation 阶段从内部 round-trip 验证推进到 EasyCAT reference validation。
- Issues [#7](https://github.com/guajun/ethercat-configurator/issues/7)、[#8](https://github.com/guajun/ethercat-configurator/issues/8)、[#9](https://github.com/guajun/ethercat-configurator/issues/9) 已 reopen，并更新为 PR #19 的当前目标。
- 新增 issue [#18](https://github.com/guajun/ethercat-configurator/issues/18) 跟踪 firmware header 的 `__packed` switch；它不属于 PR #19 范围。
- EasyCAT reference fixtures 采用一目录一套配对规则：`device.yaml`、`.prj`、`.xml`、`.bin`、`.h` 必须同目录维护，避免 device 与 EasyCAT project 漂移。

## 产品边界

构建一个 CLI-first 的 EtherCAT SubDevice 配置生成器和校验器。

本项目负责：

- 声明式设备配置。
- Object Dictionary 和 PDO layout 校验。
- ESI XML 生成和检查。
- SII EEPROM binary 生成和检查。
- 面向 process data offset 的固件头文件生成。
- 面向 CI 和 agent workflow 的确定性报告。

早期不负责：

- 实时 EtherCAT master。
- 完整固件 runtime stack。
- GUI 配置器。
- 重新分发受限的 ETG 文档或 schema。

## Roadmap

### 1. CLI Foundation

建立可测试、可脚本化的 CLI 外壳：命令路由、`--help`、`--json`、`--version`、稳定 exit code 和结构化 diagnostics。这个阶段不依赖 EtherCAT 硬件。

- [#1](https://github.com/guajun/ethercat-configurator/issues/1) FEAT-000: Go CLI Skeleton
- [#3](https://github.com/guajun/ethercat-configurator/issues/3) FEAT-002: Diagnostics and JSON Output

### 2. Device Model and Layout

定义单一 `device.yaml` source of truth，并从中计算 Object Dictionary、PDO layout、Sync Manager size 和 process data address map。这里解决 EasyCAT-style hardcoded IN/OUT 地址在长 PDO 下可能重叠的问题。

- [#2](https://github.com/guajun/ethercat-configurator/issues/2) FEAT-001: Device Model and YAML Loader
- [#4](https://github.com/guajun/ethercat-configurator/issues/4) FEAT-003: PDO Layout Engine
- [#5](https://github.com/guajun/ethercat-configurator/issues/5) FEAT-004: Process Data Address Map

### 3. Reviewable Outputs

先生成可读、可 diff 的报告，让配置变更可以在 PR 中被审阅，也让 AI agent 和 CI 能拿到结构化输出。

- [#6](https://github.com/guajun/ethercat-configurator/issues/6) FEAT-005: Reports

### 4. Artifact Generation

从同一个 canonical model 生成 EtherCAT 工程产物和固件集成文件，保证 XML、EEPROM binary、header 使用同一份 layout/address 结果。

- [#7](https://github.com/guajun/ethercat-configurator/issues/7) FEAT-006: Minimal ESI XML Generation
- [#8](https://github.com/guajun/ethercat-configurator/issues/8) FEAT-007: SII EEPROM Binary Generation
- [#9](https://github.com/guajun/ethercat-configurator/issues/9) FEAT-008: Firmware Header Generation
- [#18](https://github.com/guajun/ethercat-configurator/issues/18) Follow-up: `__packed` switch for firmware header

### 5. Consistency and References

检查声明式配置与生成/导入产物是否漂移，并用安全的 metadata 方式追踪标准文档引用，不提交受限 ETG 资产。

- [#10](https://github.com/guajun/ethercat-configurator/issues/10) FEAT-009: Diff and Artifact Consistency
- [#11](https://github.com/guajun/ethercat-configurator/issues/11) FEAT-010: Specs Index

### 6. Optional Online Verification

在离线生成器稳定后，再通过外部工具或硬件扫描结果做可选验证。核心项目仍然必须在没有硬件时可用。

- [#12](https://github.com/guajun/ethercat-configurator/issues/12) FEAT-011: Online Verification Adapter

## Issue Index

| Issue | Feature | Source of Truth |
| --- | --- | --- |
| [#1](https://github.com/guajun/ethercat-configurator/issues/1) | FEAT-000: Go CLI Skeleton | Issue #1 |
| [#2](https://github.com/guajun/ethercat-configurator/issues/2) | FEAT-001: Device Model and YAML Loader | Issue #2 |
| [#3](https://github.com/guajun/ethercat-configurator/issues/3) | FEAT-002: Diagnostics and JSON Output | Issue #3 |
| [#4](https://github.com/guajun/ethercat-configurator/issues/4) | FEAT-003: PDO Layout Engine | Issue #4 |
| [#5](https://github.com/guajun/ethercat-configurator/issues/5) | FEAT-004: Process Data Address Map | Issue #5 |
| [#6](https://github.com/guajun/ethercat-configurator/issues/6) | FEAT-005: Reports | Issue #6 |
| [#7](https://github.com/guajun/ethercat-configurator/issues/7) | FEAT-006: Minimal ESI XML Generation | Issue #7 |
| [#8](https://github.com/guajun/ethercat-configurator/issues/8) | FEAT-007: SII EEPROM Binary Generation | Issue #8 |
| [#9](https://github.com/guajun/ethercat-configurator/issues/9) | FEAT-008: Firmware Header Generation | Issue #9 |
| [#10](https://github.com/guajun/ethercat-configurator/issues/10) | FEAT-009: Diff and Artifact Consistency | Issue #10 |
| [#11](https://github.com/guajun/ethercat-configurator/issues/11) | FEAT-010: Specs Index | Issue #11 |
| [#12](https://github.com/guajun/ethercat-configurator/issues/12) | FEAT-011: Online Verification Adapter | Issue #12 |
| [#18](https://github.com/guajun/ethercat-configurator/issues/18) | Follow-up: `__packed` switch for firmware header | Issue #18 |

## 初始仓库布局

```text
cmd/ethercat-configurator/
internal/adapter/
internal/config/
internal/coe/
internal/diag/
internal/esi/
internal/firmware/
internal/model/
internal/pdo/
internal/report/
internal/sii/
schemas/
specs/
fixtures/
examples/
```

## Fixture 策略

- `examples/lan9252-basic`: 最小可用 SubDevice，包含一个 RX PDO 和一个 TX PDO。
- `fixtures/invalid`: 格式错误或语义非法的 YAML case。
- `fixtures/pdo`: packed bits、byte-aligned entries、mixed-width integers 和 padding case。
- `fixtures/pdi`: PDI 地址大小、alignment、range 和显式地址覆盖 case。
- `fixtures/esi`: 生成的 XML golden files，以及 license 允许的导入 XML samples。
- `fixtures/sii`: 生成的 binary golden files，以及故意破坏 checksum 的 case。
- `fixtures/easycat`: EasyCAT oracle fixtures。每个子目录必须将 `device.yaml` 与同名 `.prj` 及其生成/修正后的 `.xml`、`.bin`、`.h` 放在一起。

当前 EasyCAT oracle cases：

- `fixtures/easycat/easycat_safe_64`: EasyCAT 默认 `0x1000`/`0x1200` 在 64-byte triple-buffer layout 下安全。
- `fixtures/easycat/ethercat_configurator_reference`: 192-byte fixed reference，显式使用 `0x1000`/`0x1800` 避免 overlap；XML 和 binary 都必须体现 `0x1800`。
- `fixtures/easycat/known_bad_overlap`: EasyCAT 默认 `0x1000`/`0x1200` 的 192-byte known-bad reference；生成器应输出明确 warning，但不直接拒绝产物生成。

## 命令面

```bash
ethercat-configurator validate device.yaml
ethercat-configurator validate --help
ethercat-configurator validate specs
ethercat-configurator report device.yaml -o report.md
ethercat-configurator report device.yaml --json
ethercat-configurator gen esi device.yaml -o device.xml
ethercat-configurator gen esi --help
ethercat-configurator gen sii device.yaml -o eeprom.bin
ethercat-configurator gen header device.yaml -o ethercat_device.h
ethercat-configurator inspect esi device.xml
ethercat-configurator inspect sii eeprom.bin
ethercat-configurator diff expected.yaml actual.xml
ethercat-configurator diff expected.yaml eeprom.bin
```

## 开发规则

- 一个 canonical intermediate model 供给所有生成产物。
- 具体任务 scope、验收标准和讨论以 GitHub Issues 为准。
- `PLAN.md` 只维护阶段顺序、issue 索引和跨 issue 的设计约束。
- 默认生成结果必须确定。
- CLI 命令不能要求交互式输入。
- 面向机器的命令在有意义时都应支持 JSON。
- Diagnostics 必须使用稳定 code 和确定性排序。
- 测试应从 package-local 开始，只在输出刻意稳定时添加 golden files。
- 受限 ETG documents、ZIP files 和 schemas 不能被 vendored。

## 测试策略

- model validation 和 type handling 的单元测试。
- YAML 到 report、YAML 到 ESI XML、YAML 到 SII binary 的 golden tests。
- 生成 SII binaries 的 round-trip tests。
- 在 license 允许时，导入 known-good simple devices 作为 compatibility fixtures。
- Optional hardware tests 通过环境变量开启。

## 发布策略

Go distribution targets：

- Windows amd64/arm64
- Linux amd64/arm64
- macOS amd64/arm64

Release assets：

- Static CLI binaries。
- Checksums。
- 可行时提供 SBOM。
- 示例生成产物。

## 下一步

1. Merge PR [#19](https://github.com/guajun/ethercat-configurator/pull/19) 到 `develop`，完成 EasyCAT reference validation 和 explicit buffer-mode warning 这一轮。
2. 继续 [#10](https://github.com/guajun/ethercat-configurator/issues/10)，把声明式配置与导入/生成产物的 diff consistency 做成显式命令。
3. 继续 [#11](https://github.com/guajun/ethercat-configurator/issues/11)，维护 specs index，但不 vendored 受限 ETG assets。
4. 评估 [#18](https://github.com/guajun/ethercat-configurator/issues/18) 的 `__packed` switch，作为 firmware header 的独立后续小改动。
