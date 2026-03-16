extern crate zed_extension_api;
use std::fs;
use zed::LanguageServerId;
use zed_extension_api::{self as zed, Result};

struct DesignTokensExtension {
    cached_binary_path: Option<String>,
}

impl DesignTokensExtension {
    fn language_server_binary(
        &mut self,
        language_server_id: &LanguageServerId,
        worktree: &zed::Worktree,
    ) -> Result<String> {
        if let Some(path) = worktree.which("asimonim") {
            return Ok(path);
        }

        if let Some(path) = &self.cached_binary_path {
            if fs::metadata(path).map_or(false, |stat| stat.is_file()) {
                return Ok(path.clone());
            }
        }

        zed::set_language_server_installation_status(
            language_server_id,
            &zed::LanguageServerInstallationStatus::CheckingForUpdate,
        );
        let release = match zed::latest_github_release(
            "bennypowers/asimonim",
            zed::GithubReleaseOptions {
                require_assets: true,
                pre_release: false,
            },
        ) {
            Ok(release) => release,
            Err(err) => {
                // Fall back to cached binary if GitHub API fails
                if let Some(path) = &self.cached_binary_path {
                    if fs::metadata(path).map_or(false, |stat| stat.is_file()) {
                        return Ok(path.clone());
                    }
                }
                return Err(err);
            }
        };

        let (platform, arch) = zed::current_platform();
        // Binary names for asimonim:
        //  * - asimonim-darwin-arm64
        //  * - asimonim-darwin-x64
        //  * - asimonim-linux-arm64
        //  * - asimonim-linux-x64
        //  * - asimonim-win32-arm64.exe
        //  * - asimonim-win32-x64.exe
        let arch_name = match arch {
            zed::Architecture::Aarch64 => "arm64",
            zed::Architecture::X8664 => "x64",
            zed::Architecture::X86 => {
                return Err(format!(
                    "Unsupported architecture: 32-bit x86 is not supported."
                ))
            }
        };
        let (os_name, ext) = match platform {
            zed::Os::Mac => ("darwin", ""),
            zed::Os::Linux => ("linux", ""),
            zed::Os::Windows => ("win32", ".exe"),
        };
        let asset_name = format!("asimonim-{}-{}{}", os_name, arch_name, ext);

        let asset = release
            .assets
            .iter()
            .find(|asset| asset.name == asset_name)
            .ok_or_else(|| format!("no asset found matching {:?}", asset_name))?;

        let version_dir = format!("asimonim-{}", release.version);
        fs::create_dir_all(&version_dir)
            .map_err(|err| format!("failed to create directory '{version_dir}': {err}"))?;

        let binary_path = format!("{version_dir}/{asset_name}");

        if !fs::metadata(&binary_path).map_or(false, |stat| stat.is_file()) {
            zed::set_language_server_installation_status(
                language_server_id,
                &zed::LanguageServerInstallationStatus::Downloading,
            );

            zed::download_file(
                &asset.download_url,
                &binary_path,
                zed::DownloadedFileType::Uncompressed,
            )
            .map_err(|err| format!("failed to download file: {err}"))?;

            zed::make_file_executable(&binary_path)?;

            // Clean up old version directories
            let entries = fs::read_dir(".")
                .map_err(|err| format!("failed to list working directory {err}"))?;
            for entry in entries {
                let entry = entry.map_err(|err| format!("failed to load directory entry {err}"))?;
                if let Some(name) = entry.file_name().to_str() {
                    if name != version_dir
                        && name.starts_with("asimonim-")
                        && entry.file_type().map_or(false, |ft| ft.is_dir())
                    {
                        fs::remove_dir_all(entry.path()).ok();
                    }
                }
            }
        }

        self.cached_binary_path = Some(binary_path.clone());
        Ok(binary_path)
    }
}

impl zed::Extension for DesignTokensExtension {
    fn new() -> Self {
        Self {
            cached_binary_path: None,
        }
    }

    fn language_server_command(
        &mut self,
        language_server_id: &LanguageServerId,
        worktree: &zed::Worktree,
    ) -> Result<zed::Command> {
        let binary = self.language_server_binary(language_server_id, worktree)?;
        Ok(zed::Command {
            command: binary,
            args: vec!["lsp".to_string()],
            env: Default::default(),
        })
    }
}

zed::register_extension!(DesignTokensExtension);
