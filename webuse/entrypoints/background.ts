export default defineBackground(() => {
  console.log('Auracrab Browser Extension Started', { id: browser.runtime.id });

  let socket: WebSocket | null = null;

  function connect() {
    socket = new WebSocket('ws://localhost:9999/ws');

    socket.onopen = () => {
      console.log('Connected to Auracrab backend');
    };

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
    // Basic command parser
    if (command.startsWith('open ')) {
      const url = command.substring(5);
      await browser.tabs.create({ url });
    } else if (command === 'scrape') {
      const tabs = await browser.tabs.query({ active: true, currentWindow: true });
      if (tabs[0]?.id) {
        const results = await browser.scripting.executeScript({
          target: { tabId: tabs[0].id },
          func: () => document.body.innerText,
        });
        socket?.send(JSON.stringify({
          type: 'response',
          content: results[0].result,
          id,
        }));
      }
    }
  }

  connect();
});
