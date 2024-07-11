import os
from discord.ext import commands


class handler:
    async def setup(self, client: commands.Bot):
        for file in os.listdir(f"{os.getcwd()}/astra/src/cogs"):
            if os.path.isdir(f"{os.getcwd()}/astra/src/cogs/{file}"):
                return
            print(f"Cog {file} loaded.")
            if file.endswith(".py"):
                await client.load_extension(f"cogs.{file[:-3]}")

