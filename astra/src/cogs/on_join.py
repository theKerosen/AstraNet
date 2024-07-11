import json
import os
from discord.ext import commands
from discord import Member


class Join(commands.Cog):
    def __init__(self, bot) -> commands.Bot:
        self.bot = bot

    @commands.Cog.listener()
    async def on_member_join(self, member: Member):
        file = open(f"{os.getcwd()}/config.json")
        data = json.load(file)
        await member.add_roles(member.guild.get_role(data["r_joinrole"]))

async def setup(bot: commands.Bot):
    await bot.add_cog(Join(bot))
