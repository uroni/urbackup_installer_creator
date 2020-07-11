# SPDX-License-Identifier: AGPL-3.0-or-later
from tornado.wsgi import WSGIContainer
from tornado.httpserver import HTTPServer
from tornado.ioloop import IOLoop
from app import app

http_server = HTTPServer(WSGIContainer(app))
http_server.listen(5000)
http_server.start(10)
IOLoop.instance().start()