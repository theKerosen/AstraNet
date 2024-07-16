import requests
import json
import os
import discord
from discord.ext import commands
from utils.dictionaries import api_status_dictionary
import asyncio
import threading


class loader:
    def __init__(self, bot: commands.Bot):
        self.bot = bot

    def setup_services(self) -> requests.Response:
        response = None

        try:
            api = requests.get("http://localhost:3000/app/730/status")
            response = api.json()["data"]["result"]
        except (json.JSONDecodeError, requests.exceptions.Timeout):
            print("Received an invalid response from the Ares API.")
            return None
        except api.status_code != 200:
            print(f"{api_status_dictionary[response.status_code]}")
            return None
        return response

    async def embed_message(self, channel_id, content, title, description):
        channel = self.bot.get_channel(channel_id)
        if not channel:
            return
        embed = discord.Embed(
            title=title, description=description, color=discord.Color.red()
        )
        await channel.send(content=content, embed=embed)

    def main_loader(self, channel_id):
        async def loop():
            while True:
                with open(f"{os.getcwd()}/astra/state.json", "r+") as f:
                    open_state = json.load(f)
                    state = open_state["state"]

                    response = self.setup_services()

                    if (
                        response is None
                        or response["services"] == "unknown"
                        and response["matchmaking"] == "unknown"
                    ):
                        await self.embed_message(
                            channel_id,
                            "The following services are offline:",
                            "API",
                            "The Steam API is offline.",
                        )
                    else:
                        state["sessions_logon"] = response["services"]["SessionsLogon"]
                        state["community"] = response["services"]["SteamCommunity"]
                        state["matchmaker"] = response["matchmaking"]["scheduler"]

                        f.seek(0)
                        json.dump(open_state, f, indent=4)
                        f.truncate()

                await asyncio.sleep(2)

        threading.Thread(target=asyncio.run, args=(loop(),)).start()
