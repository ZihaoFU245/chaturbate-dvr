# config package

Purpose: Build the global runtime configuration (`*entity.Config`) from CLI flags provided by `urfave/cli/v2`.

Main function:
- `New(c *cli.Context) (*entity.Config, error)`
  - Reads flags like `--username`, `--framerate`, `--resolution`, `--pattern`, `--max-duration`, `--max-filesize`, `--port`, `--interval`, `--cookies`, `--user-agent`, `--domain`.
  - Embeds the app version from `c.App.Version`.

Used by:
- `main.start` to initialize `server.Config`.

Notes:
- Values in `entity.Config` are later consumed by multiple packages (HTTP headers in `internal`, web port in `router`, intervals in `channel`, etc.).