# Codex Network Doctor MasterSpec（Blueprint v1）

> **Single Source of Truth（唯一权威文档）**

## 项目使命

构建一个企业级 Windows 网络取证（Forensics）工具。

核心定位：

- 只采集证据（Evidence）
- 不执行修复（Repair）
- 不进行推断（Inference）
- 输出可重复、可验证、可审计的 Evidence ZIP

## 核心原则

1. Evidence over inference
2. Read Only
3. Deterministic（相同环境得到相同结果）
4. Collector Isolation（采集器互不影响）
5. Raw Evidence First（优先保存原始证据）
6. Fail Open（单个 Collector 失败不影响其它）
7. Every conclusion must be backed by evidence.

## 工程资产

项目应包含：

- RFC（设计原则）
- SDD（系统设计）
- ADR（架构决策）
- JSON Schema
- Collector Catalog
- Test Handbook
- GitHub Issues
- Mermaid 架构图
- 示例 Evidence ZIP
- Contributor Guide

## 模块

- Collector Engine
- Evidence Store
- Manifest Builder
- ZIP Builder
- Validation Engine
- Timeline Engine
- GUI
- CLI

## Collector 分类

- System
- Adapter
- Route
- DNS
- Socket
- TLS
- WebSocket
- WinHTTP
- Proxy
- PAC
- Firewall
- WFP
- TUN
- v2ray
- Cockpit
- Codex
- Process
- EventLog

每个 Collector 必须定义：

- 输入
- 输出
- 权限
- 超时
- 错误码
- JSON Schema
- 单元测试
- Mock
- 性能目标
- 验收标准

## Evidence ZIP

必须包含：

- manifest.json
- timeline.json
- collectors/
- evidence/
- validation/
- logs/

不得包含：

- Diagnosis
- Root Cause
- Probability
- AI Guess

## 测试

发布前必须通过：

- go test ./...
- go test -race
- go vet
- golangci-lint
- Regression Suite
- ZIP Integrity Check

## GitHub 管理

采用：

- Issue 驱动开发
- Milestone 管理
- PR Review
- Definition of Done
- CODEOWNERS
- Conventional Commits

## Roadmap

v4：Evidence Engine

v4.1：Windows API Collectors

v4.2：GUI

v5：Plugin SDK

v6：Cross Platform

## 最终目标

本项目不是一个“网络修复器”。

它是一个**网络法医平台（Network Forensics Platform）**。

AI 的职责是分析证据。

工具的职责是采集证据。

整个项目以本文件作为唯一权威设计文档，后续 RFC、SDD、Issue 与代码均应保持一致。

---

该文件保存用户提供的 Blueprint v1 原文；当前规范性版本是 `docs/MASTER_SPEC.md`。
