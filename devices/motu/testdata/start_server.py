import asyncio
from motu_server import server


def main():
    asyncio.run(server.main())


if __name__ == "__main__":
    main()
