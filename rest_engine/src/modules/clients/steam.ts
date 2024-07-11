import SteamUser from "steam-user";
import config from "../../config.json";

//
// Steam Client
// This class runs a headless Steam Client.
//

class SteamClient extends SteamUser {
  constructor() {
    super({
      enablePicsCache: true,
      picsCacheAll: true,
      changelistUpdateInterval: 5000,
    });

    super.logOn({ anonymous: true });
    super.gamesPlayed([730], true);
    super.on("appLaunched", (e) => console.log(e));
    console.log("Steam client online.");
  }
}

export default new SteamClient();
