import win32evtlog
import win32evtlogutil
import socket
import requests
import json
import time
import os

BACKEND_URL = "http://localhost:8080/ingest"
HOSTNAME = socket.gethostname()

def send_event(event_data):
    """Sends a formatted security event to the backend API."""
    try:
        response = requests.post(BACKEND_URL, json=event_data, timeout=5)
        if response.status_code == 200:
            print(f"Successfully sent event: {event_data['details']}")
        else:
            print(f"Backend returned non-OK status: {response.status_code}")
    except requests.exceptions.RequestException as e:
        print(f"Error sending event to backend: {e}")

def main():
    """Main function to poll the Windows Security Event Log for new process creation events."""
    print(f"Agent started on host: {HOSTNAME}. Watching for process creation events... (Press Ctrl+C to exit)")

    log_type = 'Security'
    # Open the Security event log
    hand = win32evtlog.OpenEventLog(None, log_type)
    
    # Get the number of records to start from the end of the log
    total_records = win32evtlog.GetNumberOfEventLogRecords(hand)
    last_record_processed = total_records

    flags = win32evtlog.EVENTLOG_BACKWARDS_READ | win32evtlog.EVENTLOG_SEQUENTIAL_READ
    
    while True:
        try:
            # Read all new events since our last check
            events = win32evtlog.ReadEventLog(hand, flags, last_record_processed)
            
            if events:
                for event in events:
                    # We are only interested in process creation events
                    if event.EventID == 4688:
                        
                        # Data for event 4688 is in the 'StringInserts' field
                        # Indices can vary slightly, but these are common for process creation
                        try:
                            parent_process_path = event.StringInserts[12]
                            new_process_path = event.StringInserts[5]

                            # Extract just the executable name
                            parent_process_name = os.path.basename(parent_process_path)
                            new_process_name = os.path.basename(new_process_path)
                            
                            details = f"Process '{new_process_name}' launched by '{parent_process_name}'"
                            print(f"DETECTED Process Creation: {details}")

                            security_event = {
                                "timestamp": time.strftime('%Y-%m-%dT%H:%M:%SZ', time.gmtime()),
                                "hostname": HOSTNAME,
                                "event_type": "Process Creation",
                                "details": details,
                            }
                            send_event(security_event)
                        
                        except IndexError:
                            # Skip event if data is not in the expected format
                            continue
                
                # Update the last record number to the latest one we've seen
                if len(events) > 0:
                    last_record_processed = events[0].RecordNumber

            # Wait for 2 seconds before polling again
            time.sleep(2)
        
        except Exception as e:
            print(f"An error occurred: {e}. Retrying in 5 seconds...")
            time.sleep(5)

if __name__ == "__main__":
    main()