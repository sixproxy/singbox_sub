# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go application that generates sing-box proxy configuration files from subscription URLs. It fetches proxy node information from remote subscriptions, parses various proxy protocols (Shadowsocks, Hysteria2), and generates a complete sing-box configuration with routing rules, DNS settings, and organized outbound connections.

## Build and Run Commands

```bash
# Build the application
go build -o singbox_sub src/github.com/sixproxy/sub.go

# Run the application (generates singbox_config.json)
go run src/github.com/sixproxy/sub.go

# Run from built binary
./singbox_sub
```

## Architecture Overview

### Core Components

- **Main Application** (`src/github.com/sixproxy/sub.go`): Entry point that orchestrates the configuration generation process
- **Configuration Models** (`src/github.com/sixproxy/model/`): Data structures for sing-box configuration and subscription management
- **Protocol Parsers** (`src/github.com/sixproxy/protocol/`): Pluggable parsers for different proxy protocols (SS, Hysteria2)
- **Constants** (`src/github.com/sixproxy/constant/`): Shared constants for outbound types and templates

### Configuration Flow

1. **Template Loading**: Loads base configuration from `src/github.com/sixproxy/config/sbv1.12.json`
2. **Subscription Fetching**: Downloads and decodes base64-encoded proxy node lists from subscription URLs
3. **Node Parsing**: Parses proxy URLs using protocol-specific parsers (concurrent processing)
4. **Template Rendering**: Applies node filtering and populates outbound configurations
5. **Output Generation**: Writes final sing-box configuration to `singbox_config.json`

### Key Design Patterns

- **Plugin Architecture**: Protocol parsers implement the `Parser` interface and auto-register via `init()` functions
- **Template System**: Uses `{all}` placeholders and filtering rules to dynamically populate outbound lists
- **Factory Pattern**: `NewOutbound()` creates appropriate outbound configuration types based on protocol
- **Concurrent Processing**: Node parsing runs in parallel using goroutines and channels

## Configuration Template Structure

The base template (`src/github.com/sixproxy/config/sbv1.12.json`) contains:

- **Subscription Definition**: URL and settings for fetching proxy nodes
- **Outbound Templates**: Selector and urltest groups with `{all}` placeholders
- **Filtering Rules**: Include/exclude patterns for organizing nodes by region or type
- **Routing Rules**: Traffic routing based on domain, GeoIP, and application-specific rules
- **DNS Configuration**: Split DNS setup with local and proxy resolvers

## Key Files and Their Purpose

- `src/github.com/sixproxy/sub.go`: Main application entry point and orchestration
- `src/github.com/sixproxy/model/config.go`: Core configuration structure and template rendering logic
- `src/github.com/sixproxy/model/factory.go`: Outbound configuration factory
- `src/github.com/sixproxy/protocol/parse.go`: Protocol parser registry and interface
- `src/github.com/sixproxy/protocol/ss.go`: Shadowsocks protocol parser implementation
- `src/github.com/sixproxy/protocol/hysteria2.go`: Hysteria2 protocol parser implementation

## Adding New Protocol Support

1. Create parser file in `src/github.com/sixproxy/protocol/`
2. Implement the `Parser` interface with `Proto()` and `Parse()` methods
3. Register parser in `init()` function: `parsers["protocol"] = &yourParser{}`
4. Add protocol constant to `src/github.com/sixproxy/constant/outbound.go`
5. Add outbound model in `src/github.com/sixproxy/model/`
6. Update factory in `src/github.com/sixproxy/model/factory.go`

## Node Filtering System

The template uses filtering rules to organize nodes:

- **Include filters**: `"action": "include", "keywords": ["日本"]` - only nodes matching these patterns
- **Exclude filters**: `"action": "exclude", "keywords": ["官网|流量"]` - exclude nodes matching these patterns
- **Pattern support**: Supports both simple string matching and regex patterns
- **Multiple patterns**: Use `|` separator for OR conditions

## Development Guidelines

- Configuration changes should be made in the template file (`src/github.com/sixproxy/config/sbv1.12.json`)
- Protocol parsers must handle URL parsing errors gracefully
- New outbound types require updates to both the model and factory
- Node filtering supports both string contains and regex matching
- All JSON marshaling/unmarshaling uses custom methods to handle interface types