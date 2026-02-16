export default defineBackground(() => {
  console.log('Auracrab Browser Extension Started', { id: browser.runtime.id });

  let socket: WebSocket | null = null;

  function connect() {
    socket = new WebSocket('ws://localhost:9999/ws');

    socket.onopen = async () => {
      console.log('Connected to Auracrab backend');
      await register();
    };

    async function register() {
      // Get or create a unique instance ID for this browser installation
      let { instanceId } = await browser.storage.local.get('instanceId');
      if (!instanceId) {
        instanceId = crypto.randomUUID();
        await browser.storage.local.set({ instanceId });
      }

      const window = await browser.windows.getCurrent();
      const tabs = await browser.tabs.query({ windowId: window.id });
      const info = await browser.runtime.getPlatformInfo();
      
      socket?.send(JSON.stringify({
        type: 'register',
        profile: `browser-${info.os}`,
        instanceId: instanceId,
        windowId: window.id?.toString(),
        tabs: tabs.map(t => ({ id: t.id, url: t.url, title: t.title }))
      }));
    }

    // Update registration info when tabs change
    browser.tabs.onUpdated.addListener(() => register());
    browser.tabs.onRemoved.addListener(() => register());
    browser.tabs.onCreated.addListener(() => register());


    socket.onmessage = async (event) => {
      try {
        const data = JSON.parse(event.data);
        console.log('Received command:', data);

        if (data.type === 'command') {
          await handleCommand(data.content, data.id);
        }
      } catch (err) {
        console.error('Failed to process message:', err);
      }
    };

    socket.onclose = () => {
      console.log('Disconnected from Auracrab backend, retrying in 5s...');
      setTimeout(connect, 5000);
    };

    socket.onerror = (err) => {
      console.error('WebSocket error:', err);
      socket?.close();
    };
  }

  async function handleCommand(command: string, id?: string) {
    try {
      if (command.startsWith('open ')) {
        const url = command.substring(5);
        await browser.tabs.create({ url });
        sendResponse(id, "Opened " + url);
      } else if (command === 'scrape') {
        const results = await executeInActiveTab(() => document.body.innerText);
        sendResponse(id, results[0]?.result || "");
      } else if (command.startsWith('click ')) {
        const selector = command.substring(6);
        await executeInActiveTab((sel) => {
          const el = document.querySelector(sel) as HTMLElement;
          if (el) {
            el.scrollIntoView({ behavior: 'smooth', block: 'center' });
            el.click();
            return "Clicked " + sel;
          }
          return "Element not found: " + sel;
        }, [selector]);
        sendResponse(id, "Click command sent for " + selector);
      } else if (command.startsWith('type ')) {
        const parts = command.substring(5).split(' ');
        const selector = parts[0];
        const text = parts.slice(1).join(' ');
        await executeInActiveTab((sel, txt) => {
          const el = document.querySelector(sel) as HTMLInputElement;
          if (el) {
            el.scrollIntoView({ behavior: 'smooth', block: 'center' });
            el.focus();
            el.value = txt;
            el.dispatchEvent(new Event('input', { bubbles: true }));
            el.dispatchEvent(new Event('change', { bubbles: true }));
            return "Typed into " + sel;
          }
          return "Element not found: " + sel;
        }, [selector, text]);
        sendResponse(id, "Type command sent for " + selector);
      }
    } catch (err) {
      sendResponse(id, "Error: " + String(err));
    }
  }

  function sendResponse(id: string | undefined, content: string) {
    if (id && socket && socket.readyState === WebSocket.OPEN) {
      socket.send(JSON.stringify({ type: 'response', content, id }));
    }
  }

  async function executeInActiveTab<T>(func: (...args: any[]) => T, args: any[] = []): Promise<any[]> {
    const tabs = await browser.tabs.query({ active: true, currentWindow: true });
    if (!tabs[0]?.id) throw new Error("No active tab");
    return browser.scripting.executeScript({
      target: { tabId: tabs[0].id },
      func,
      args,
    });
  }

  connect();
});
