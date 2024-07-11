from utils.load import loader
from utils.services import cs_services
from utils.changenumber_check import changenumber_check
import asyncio


class thread:
    def __init__(self, bot, channel_id, role_id):
        self.bot = bot
        self.channel_id = channel_id
        self.role_id = role_id

    async def setup(self):
        await self._setup()

    async def _setup(self):
        try:
            await self.load_services()
            await self.check_services()
        except TypeError as e:
            print(e)

    async def load_services(self):
        load = loader(self.bot)
        load.main_loader(self.channel_id)

    async def check_services(self):
        await changenumber_check(self.bot).check_changenumber()
        await cs_services(self.bot).check_services(self.channel_id, self.role_id)

