#!/usr/bin/env bash
# Builds dokku-mcp.mcpb, the MCP Bundle listed in the official MCP Registry.
#
# The bundle carries the release binaries for every supported platform and a
# small launcher that starts the one matching the host, so a single package
# serves macOS and Linux on amd64, arm64 and arm.
#
# Usage: scripts/build-mcpb.sh <version> <binaries-dir> <output-file>
#   version       release version, with or without the leading "v"
#   binaries-dir  directory holding dokku-mcp-<os>-<arch> release binaries
#   output-file   path of the .mcpb file to write
set -euo pipefail

if [[ $# -ne 3 ]]; then
  echo "usage: $0 <version> <binaries-dir> <output-file>" >&2
  exit 2
fi

version="${1#v}"
bin_dir="$2"
output="$3"
platforms=(darwin-amd64 darwin-arm64 linux-amd64 linux-arm64 linux-arm)

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
mkdir -p "$work/server"

for platform in "${platforms[@]}"; do
  src="$bin_dir/dokku-mcp-$platform"
  if [[ ! -f "$src" ]]; then
    echo "missing binary: $src" >&2
    exit 1
  fi
  install -m 0755 "$src" "$work/server/dokku-mcp-$platform"
done

cat > "$work/server/launch.sh" <<'EOF'
#!/bin/sh
# Starts the dokku-mcp binary built for this machine.
set -e
dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
os=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$(uname -m)" in
  x86_64 | amd64) arch=amd64 ;;
  aarch64 | arm64) arch=arm64 ;;
  armv6* | armv7*) arch=arm ;;
  *) echo "dokku-mcp: unsupported architecture $(uname -m)" >&2; exit 1 ;;
esac
bin="$dir/dokku-mcp-$os-$arch"
if [ ! -f "$bin" ]; then
  echo "dokku-mcp: no binary for $os/$arch in this bundle" >&2
  exit 1
fi
# Bundle extraction does not always keep the executable bit.
[ -x "$bin" ] || chmod +x "$bin"
exec "$bin" "$@"
EOF
chmod 0755 "$work/server/launch.sh"

cp LICENSE "$work/LICENSE"

jq -n --arg version "$version" '{
  manifest_version: "0.3",
  name: "dokku-mcp",
  display_name: "Dokku MCP Server",
  version: $version,
  description: "Manage Dokku apps, deployments, datastores, domains and HTTPS over SSH.",
  long_description: "Lets an AI assistant create, deploy, scale, configure and inspect apps on a Dokku server. dokku-mcp runs locally and talks to Dokku over SSH, using your ssh-agent or the key you choose.",
  author: { name: "dokku-MCP", url: "https://github.com/dokku-MCP" },
  repository: { type: "git", url: "https://github.com/dokku-MCP/dokku-mcp" },
  homepage: "https://github.com/dokku-MCP/dokku-mcp",
  documentation: "https://github.com/dokku-MCP/dokku-mcp#readme",
  support: "https://github.com/dokku-MCP/dokku-mcp/issues",
  license: "Apache-2.0",
  keywords: ["dokku", "paas", "deployment", "ssh", "devops"],
  server: {
    type: "binary",
    entry_point: "server/launch.sh",
    mcp_config: {
      command: "/bin/sh",
      args: ["${__dirname}/server/launch.sh"],
      env: {
        DOKKU_MCP_SSH_HOST: "${user_config.ssh_host}",
        DOKKU_MCP_SSH_PORT: "${user_config.ssh_port}",
        DOKKU_MCP_SSH_USER: "${user_config.ssh_user}",
        DOKKU_MCP_SSH_KEY_PATH: "${user_config.ssh_key_path}"
      }
    }
  },
  user_config: {
    ssh_host: {
      type: "string",
      title: "Dokku host",
      description: "Hostname or IP address of the Dokku server.",
      required: true
    },
    ssh_port: {
      type: "number",
      title: "SSH port",
      description: "SSH port of the Dokku server.",
      default: 22,
      min: 1,
      max: 65535
    },
    ssh_user: {
      type: "string",
      title: "SSH user",
      description: "User for Dokku commands over SSH.",
      default: "dokku"
    },
    ssh_key_path: {
      type: "file",
      title: "SSH private key",
      description: "Private key for the Dokku server. Leave empty to use ssh-agent or the keys in ~/.ssh.",
      required: false
    }
  },
  compatibility: { platforms: ["darwin", "linux"] }
}' > "$work/manifest.json"

output_abs="$(cd "$(dirname "$output")" && pwd)/$(basename "$output")"
rm -f "$output_abs"
(cd "$work" && zip -q -X -r "$output_abs" manifest.json LICENSE server)
echo "wrote $output_abs"
