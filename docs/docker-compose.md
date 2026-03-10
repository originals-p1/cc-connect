# Docker Compose Deployment

This repository ships a `Dockerfile` and `docker-compose.yml` for running `cc-connect` in a container.

The stock image includes:

- `cc-connect`
- `ffmpeg`
- `node` / `npm`
- `codex` (`@openai/codex`)

The stock image does **not** preinstall every supported agent CLI. Codex works out of the box. If you use Gemini CLI, iFlow, OpenCode, Cursor Agent, Qoder CLI, or Claude Code inside the container, install the required CLI in a derived image or extend the container at runtime.

## 1. Prepare directories

```bash
git clone https://github.com/chenhg5/cc-connect.git
cd cc-connect
mkdir -p docker-data
cp config.example.toml docker-data/config.toml
```

If your agent `work_dir` points to local repositories, mount them into the container and use the container path in `config.toml`, for example `/workspace/my-project`.

The default `docker-compose.yml` already mounts:

- `./docker-data` -> `/data`
- `${HOME}/workspace` -> `/workspace`

Adjust those paths to match your machine.

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

Example for Gemini CLI:

```dockerfile
FROM cc-connect:local

RUN npm install -g @google/gemini-cli
```

Then change `docker-compose.yml` to build from that derived image instead of the stock Dockerfile.

## 4. Start

```bash
docker compose up -d --build
```

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
- If you need proxy access, set `HTTP_PROXY` / `HTTPS_PROXY` / `ALL_PROXY` in your shell or edit `docker-compose.yml`.
- The provided `docker-compose.yml` also forwards proxy variables into the image build, so `docker compose up -d --build` works behind a proxy without extra edits.
- The provided `docker-compose.yml` uses `build.network: host`, which is useful on Linux when your proxy listens on `127.0.0.1`.
