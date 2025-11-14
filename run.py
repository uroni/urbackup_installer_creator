# SPDX-License-Identifier: AGPL-3.0-or-later
import asyncio
from tornado.wsgi import WSGIContainer
from tornado.httpserver import HTTPServer
import tornado.netutil
from app import app

sockets = tornado.netutil.bind_sockets(5000)
tornado.process.fork_processes(10)
async def post_fork_main():
    server = HTTPServer(WSGIContainer(app))
    server.add_sockets(sockets)
    await asyncio.Event().wait()

asyncio.run(post_fork_main())