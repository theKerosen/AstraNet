from utils.load import loader

class thread:
    def __init__(self, bot, channel_id, role_id):
        self.bot = bot
        self.channel_id = channel_id
        self.role_id = role_id

    async def setup(self):
        try:
            load = loader(self.bot)
            load.main_loader(self.channel_id)
            print("setup done")
        except TypeError as e:
            print(e)
