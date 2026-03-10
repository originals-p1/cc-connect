# Docker Compose Deployment

This repository ships a `Dockerfile` and `docker-compose.yml` for running `cc-connect` in a container.

The stock image includes:

- `cc-connect`
- `node` / `npm`
- `codex` (`@openai/codex`)

The stock image is intentionally minimal and only preinstalls Codex. If you use Gemini CLI, iFlow, OpenCode, Cursor Agent, Qoder CLI, Claude Code, or voice features that require `ffmpeg`, install the required tools in a derived image.

## 1. Prepare directories

```bash
git clone https://github.com/chenhg5/cc-connect.git
cd cc-connect
mkdir -p docker-data
cp config.example.toml docker-data/config.toml
export PUID=$(id -u)
export PGID=$(id -g)
```

If your agent `work_dir` points to local repositories, mount them into the container and use the container path in `config.toml`, for example `/workspace/my-project`.

The default `docker-compose.yml` already mounts:

- `./docker-data` -> `/data`
- `/home/tz/workspace` -> `/workspace`

Adjust those paths to match your machine. Use an absolute host path instead of `${HOME}` because some Docker installations rewrite `HOME` during compose interpolation.

The container runs as `${PUID}:${PGID}` by default so files created in `docker-data` and mounted workspaces stay owned by your host user instead of `root`.

## 2. Configure `docker-data/config.toml`

Use `/data` as the container data directory:

```toml
data_dir = "/data"
```

Example Codex project:

```toml
[[projects]]
name = "my-project"

[projects.agent]
type = "codex"

[projects.agent.options]
work_dir = "/workspace/my-project"
mode = "yolo"

[[projects.platforms]]
type = "telegram"

[projects.platforms.options]
token = "123456:ABC-xxx"
```

## 3. Optional: install more agent CLIs in a derived image

Codex is already included. Extend the image only when you need other agent CLIs.

Example for Gemini CLI and voice support:

```dockerfile
FROM cc-connect:local

RUN apt-get update && apt-get install -y --no-install-recommends ffmpeg \
  && rm -rf /var/lib/apt/lists/*
RUN npm install -g @google/gemini-cli
```

Then change `docker-compose.yml` to build from that derived image instead of the stock Dockerfile.

## 4. Start

```bash
docker compose up -d --build
```

If you prefer, put `PUID` and `PGID` into a local `.env` file instead of exporting them in your shell:

```dotenv
PUID=1000
PGID=1000
```

If you need a host-local proxy such as `127.0.0.1:7890`, set runtime proxy variables like this:

```dotenv
DOCKER_HTTP_PROXY=http://host.docker.internal:7890
DOCKER_HTTPS_PROXY=http://host.docker.internal:7890
DOCKER_ALL_PROXY=http://host.docker.internal:7890
```

The compose file applies these values to both uppercase and lowercase proxy variables because different CLIs and SDKs read different variants.

## 5. Logs

```bash
docker compose logs -f cc-connect
```

## 6. Upgrade

```bash
git pull
docker compose up -d --build
```

## 7. Notes

- For LINE / WeCom and other webhook-based platforms, expose the configured ports in `docker-compose.yml`.
- If you use bot mode, make sure the workspace root in `config.toml` matches a mounted container path such as `/workspace`.
- The compose service sets `HOME=/data/.home` so Codex and other CLIs keep their writable state inside the mounted data directory.
- If you need proxy access at runtime, use `DOCKER_HTTP_PROXY` / `DOCKER_HTTPS_PROXY` / `DOCKER_ALL_PROXY` and point them to `host.docker.internal` instead of `127.0.0.1`.
- The provided `docker-compose.yml` also forwards proxy variables into the image build, so `docker compose up -d --build` works behind a proxy without extra edits.
- The provided `docker-compose.yml` uses `build.network: host`, which is useful on Linux when your proxy listens on `127.0.0.1`.
