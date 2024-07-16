import asyncio
import os
import json
import discord


class depot_check:
    def __init__(self, bot):
        self.bot = bot
        self.depot_back = {}
        self.file_timestamp = None

    async def depot_check(self):
        print("depot service initialized")
        while True:
            await self.loop_check(),
            await asyncio.sleep(2)

    async def loop_check(self):
        changes_file_path = os.path.join(
            os.getcwd(), "data_engine", "bin", "730_changes.json"
        )

        try:
            current_timestamp = os.path.getmtime(changes_file_path)

            if self.file_timestamp is None or current_timestamp != self.file_timestamp:
                self.file_timestamp = current_timestamp
                await self.process_file_changes(changes_file_path)

        except FileNotFoundError:
            pass

    async def process_file_changes(self, file_path):
        with open(file_path, "r", encoding="utf-8") as file:
            changes = json.load(file)

        new_depots = changes.get("depots_new")
        old_depots = changes.get("depots_old")

        if not new_depots or not old_depots:
            return

        if not self.depot_back:
            self.depot_back = changes
            return

        updated_depots = {}

        for depot_name, new_depot_updates in new_depots.items():
            old_depot_updates = self.depot_back["depots_new"].get(depot_name)
            if old_depot_updates is None:
                updated_depots[depot_name] = new_depot_updates
                continue

            updated_manifests = {}
            for manifest_name, new_manifest_update in new_depot_updates.items():
                old_manifest_update = old_depot_updates.get(manifest_name)
                if (
                    old_manifest_update is None
                    or old_manifest_update != new_manifest_update
                ):
                    updated_manifests[manifest_name] = new_manifest_update

            if updated_manifests:
                updated_depots[depot_name] = updated_manifests

        if updated_depots:
            with open(
                f"{os.getcwd()}/config.json", "r", encoding="utf-8"
            ) as config_file:
                config = json.load(config_file)
            channel = self.bot.get_channel(config["c_depot"])

            embed = discord.Embed(
                title="Depot Update",
                description=f"Changelist {changes['latest']}\nChanged:"
                + "\n".join(
                    f" **Depot ({manifest_name}**, {manifest_update['gid']})"
                    for _, depot_updates in updated_depots.items()
                    for manifest_name, manifest_update in depot_updates.items()
                ),
                color=discord.Color.dark_purple(),
            )
            embed.set_thumbnail(
                url="https://cdn.cloudflare.steamstatic.com/steamcommunity/public/images/apps/730/8dbc71957312bbd3baea65848b545be9eae2a355.jpg"
            )
            await channel.send(content=f"<@&{config['r_depot']}>", embed=embed)

        self.depot_back.update(changes)
