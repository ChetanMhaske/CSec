import React, { useState, useEffect } from 'react';
import './App.css';

function App() {
  const [events, setEvents] = useState([]);
  const [error, setError] = useState(null);

  const fetchEvents = async () => {
    try {
      const response = await fetch('http://localhost:8000/events');
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      const data = await response.json();
      setEvents(data);
      setError(null);
    } catch (e) {
      console.error("Failed to fetch events:", e);
      setError("Failed to connect to backend. Is the API server running on port 8000?");
    }
  };

  useEffect(() => {
    fetchEvents();
    const interval = setInterval(fetchEvents, 3000);
    return () => clearInterval(interval);
  }, []);

  return (
    <div className="bg-gray-900 text-white min-h-screen p-8 font-sans">
      <header className="mb-8">
        <h1 className="text-4xl font-bold text-cyan-400">Sentinel AI</h1>
        <p className="text-lg text-gray-400">Live Security Event Feed</p>
      </header>
      
      <main>
        <div className="bg-gray-800 rounded-lg shadow-lg p-6">
          <h2 className="text-2xl font-semibold mb-4 border-b border-gray-700 pb-2">Latest Events</h2>
          
          {error && <p className="text-red-500 bg-red-900/50 p-4 rounded-md">{error}</p>}
          
          <div className="space-y-4 max-h-[60vh] overflow-y-auto pr-4">
            {events && events.length > 0 ? (
              events.map((event, index) => (
                <div key={index} className="bg-gray-700 p-4 rounded-md">
                  <div className="flex justify-between items-center">
                    <p className="font-mono text-green-400 text-sm">{event.hostname}</p>
                    <p className="text-xs text-gray-400">{new Date(event.timestamp).toLocaleString()}</p>
                  </div>
                  <p className="mt-2 text-lg">{event.details}</p>
                  <p className="text-sm text-cyan-400 mt-1">{event.event_type}</p>
                </div>
              ))
            ) : (
              !error && <p className="text-gray-500">Awaiting new events from the agent...</p>
            )}
          </div>
        </div>
      </main>
    </div>
  );
}

export default App;
