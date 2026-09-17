import ModpackInstallButton from "../browse/ModpackInstallButton";
import PackageBrowser, { isModpack } from "../browse/PackageBrowser";
import InstallButton from "./InstallButton";
import { useProfile } from "./ProfileContext";

// BrowseTab browses Thunderstore packages to install into the open profile.
export default function BrowseTab() {
  const { game, installed, openProfile } = useProfile();
  return (
    <PackageBrowser
      game={game}
      isInstalled={(id) => installed.has(id)}
      renderActions={(pkg) =>
        isModpack(pkg) ? (
          <ModpackInstallButton game={game} pkg={pkg} onInstalled={openProfile} />
        ) : (
          <InstallButton namespace={pkg.namespace} name={pkg.name} latestVersion={pkg.latest_version_number} />
        )
      }
    />
  );
}
