import os
import json
import discord
import asyncio

class changenumber_check:
    def __init__(self, bot):
        self.bot = bot
        self.file_timestamp = None
        self.c_number = None

    async def check_changenumber(self):
        print("changenumber initialized")
        while True:
                await self.loop_check(),
                await asyncio.sleep(2)

    async def loop_check(self):
        try:
            file_path = f"{os.getcwd()}/data_engine/bin/730_changes.json"
            current_timestamp = os.path.getmtime(file_path)

            if self.file_timestamp is None:
                self.file_timestamp = current_timestamp
                return

            if current_timestamp != self.file_timestamp:
                self.file_timestamp = current_timestamp

        except FileNotFoundError:
            print("File not found.")
            return
        return

    async def process_file_changes(self, file_path):
        with open(file_path, "r", encoding="utf-8") as f:
            data = json.loads(f.read())

        if self.c_number == data.get('old'):
            config = json.load(open(f"{os.getcwd()}/config.json"))
            channel = self.bot.get_channel(config["c_changenumber"])
            embed = discord.Embed(
                title="Counter-Strike 2 — Change Number",
                description=f"~~*{data['old']}*~~ → `{data['latest']}`",
                color=discord.Color.green(),
            )
            embed.set_thumbnail(
                url="https://cdn.cloudflare.steamstatic.com/steamcommunity/public/images/apps/730/8dbc71957312bbd3baea65848b545be9eae2a355.jpg"
            )
            await channel.send(content=f"<@&{config['r_changenumber']}>", embed=embed)
        self.c_number = data.get('latest')
