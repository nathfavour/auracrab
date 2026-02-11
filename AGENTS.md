# 🤖 AGENT GUIDELINES

## Build Protocols

To maintain a clean and sovereign development environment, all agents must adhere to the following build protocols:

1. **Isolation of Artifacts**: Compiled binaries and build artifacts must **never** be outputted directly into the source code directories.
2. **Designated Output**: All builds must be directed to the `bin/` directory located at the project root.
3. **Git Hygiene**: The `bin/` directory is strictly for local artifacts and must remain excluded from version control. Ensure it is listed in the `.gitignore` file (this project already includes `bin/` in its `.gitignore`).

Failure to follow these protocols can lead to repository pollution and conflicts with managed environments like Anyisland.
