import os

def convert_to_utf8(directory):
    for root, dirs, files in os.walk(directory):
        for file in files:
            if file.endswith('.ts') or file.endswith('.tsx'):
                filepath = os.path.join(root, file)
                try:
                    # Try to read as UTF-16LE first
                    with open(filepath, 'rb') as f:
                        raw = f.read()
                    
                    if raw.startswith(b'\xff\xfe'):
                        # It's UTF-16LE with BOM
                        text = raw.decode('utf-16le')
                    elif b'\x00' in raw:
                        # It's probably UTF-16LE without BOM
                        text = raw.decode('utf-16le')
                    else:
                        # It's probably already UTF-8
                        text = raw.decode('utf-8')
                    
                    # Write back as UTF-8 without BOM
                    with open(filepath, 'w', encoding='utf-8') as f:
                        f.write(text)
                    print(f"Converted {filepath}")
                except Exception as e:
                    print(f"Failed to convert {filepath}: {e}")

convert_to_utf8(r'c:\Users\yusha\Documents\HOME_DATA\HONET\Project\PEKAN\frontend\src')
