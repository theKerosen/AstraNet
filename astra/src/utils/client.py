import asyncio
import json
import os
from utils.thread import thread
from utils.handler import handler
from discord.ext import commands
from utils.services import cs_services
from utils.changenumber_check import changenumber_check
from utils.depot_check import depot_check
import discord
import datetime, time


class AstraNet(commands.Bot):
    def __init__(self, command_prefix, intents):
        super().__init__(command_prefix=command_prefix, intents=intents)

    async def on_ready(self):
        file = open(f"{os.getcwd()}/config.json")
        data = json.load(file)

        await handler().setup(self)
        print("Connected to Discord.")
        asyncio.create_task(thread(self, data["c_uptime"], data["r_uptime"]).setup())
        asyncio.create_task(cs_services(self).check_services(data["c_uptime"], data["r_uptime"]))
        asyncio.create_task(changenumber_check(self).check_changenumber())
        asyncio.create_task(depot_check(self).depot_check())
        self.remove_command("help")
        asyncio.create_task(self.presence())
    
    async def presence(self):
        timenow = time.time()
        while self.is_ready():
            try:
                uptime = str(datetime.timedelta(seconds=int(round(time.time()-timenow))))
                act = discord.CustomActivity(name=f"Uptime: {uptime}", emoji="⏲")
                await self.change_presence(activity=act)
                await asyncio.sleep(5)
            except Exception as e:
                print(e)
    def setup(self):
        file = open(f"{os.getcwd()}/config.json")
        data = json.load(file)
        self.run(data["token"])
        return self
