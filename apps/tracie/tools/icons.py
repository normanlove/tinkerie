# Renders web/icon.svg to ico/*.png using Playwright + Chrome. Run from the source folder: mkdir ico && python tools/icons.py
import asyncio
from playwright.async_api import async_playwright
svg=open('web/icon.svg').read()
async def main():
    async with async_playwright() as p:
        b = await p.chromium.launch(executable_path="/usr/bin/google-chrome", args=["--no-sandbox"])
        for s in [16,20,24,32,40,48,64,128,256]:
            pg = await b.new_page(viewport={"width":s,"height":s})
            await pg.set_content(f'<html><body style="margin:0;background:transparent">{svg.replace("<svg ", f"<svg width=\"{s}\" height=\"{s}\" ",1)}</body></html>')
            await pg.screenshot(path=f"ico/{s}.png", omit_background=True)
            await pg.close()
        await b.close()
asyncio.run(main())
