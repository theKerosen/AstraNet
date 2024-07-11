import os
import json
import discord
import asyncio

class depot_check:
    def __init__(self, bot):
        self.bot = bot
        self.file_timestamp = None

    async def loop_check(self):
        try:
            file_path = f"{os.getcwd()}/data_engine/bin/730_changes.json"
            current_timestamp = os.path.getmtime(file_path)

            if self.file_timestamp is None:
                self.file_timestamp = current_timestamp
                return

            if current_timestamp != self.file_timestamp:
                self.file_timestamp = current_timestamp
                await self.process_file_changes(file_path)

        except FileNotFoundError:
            return

    async def process_file_changes(self, file_path):
        with open(file_path, "r", encoding="utf-8") as f:
            data = json.load(f)

        if not data:
            return

        if not data["depot_updates"]:
            return

        backup_depots = {}
        backup_depots.update(data)

        for depot_name, depot_updates in backup_depots["depot_updates"].items():
            output = ""
            for manifest_name, manifest_update in depot_updates.items():
                output += f"**{manifest_name}**\n> `{manifest_update['gid']}`\n> ~~*{manifest_update['old_gid']}*~~\n"

            file = open(f"{os.getcwd()}/config.json")
            config = json.load(file)
            channel = self.bot.get_channel(config["c_depot"])

            embed = discord.Embed(
                title=f"Depot — {depot_name}",
                description=output,
                color=discord.Color.dark_purple(),
            )
            embed.set_thumbnail(
                url="https://cdn.cloudflare.steamstatic.com/steamcommunity/public/images/apps/730/8dbc71957312bbd3baea65848b545be9eae2a355.jpg"
            )
            print(f"Depot: {depot_name}")
            await channel.send(content=f"<@&{config['r_depot']}>", embed=embed)