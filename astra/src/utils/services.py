import discord
import json
import os
import asyncio
from discord.ext import commands
from utils.dictionaries import status_dictionary, titles

class cs_services:
    def __init__(self, client: commands.Bot):
        self.bot = client

    async def embed_message(
        self,
        channel_id: int,
        role_id: int,
        service: str,
        state: str,
        color: discord.Color,
    ):
        channel = self.bot.get_channel(channel_id)
        if not channel:
            return
        content = f"<@&{role_id}>"
        embed = discord.Embed(
            title=titles[service],
            description=f"The {titles[service]} service is `{status_dictionary[state]}`",
            color=color,
        )
        embed.set_thumbnail(
            url="https://cdn.cloudflare.steamstatic.com/steamcommunity/public/images/apps/730/8dbc71957312bbd3baea65848b545be9eae2a355.jpg"
        )
        await channel.send(content=content, embed=embed)

    async def check_services(self, channel_id: int, role_id: int):
        while True:
            with open(f"{os.getcwd()}/astra/state.json", "r+") as f:
                open_state = json.load(f)
                state = open_state["state"]
                last_state = open_state["last_state"]

                for service in state:
                    if state[service] != "normal" and last_state[service] != state[service]:
                        await self.embed_message(
                            channel_id,
                            role_id,
                            service,
                            state[service],
                            discord.Color.blurple(),
                        )
                    last_state[service] = state[service]

                f.seek(0)
                json.dump(open_state, f, indent=4)
                f.truncate()
            await asyncio.sleep(5)

