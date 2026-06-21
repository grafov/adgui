#!/usr/bin/env python3
import os
import argparse
import sys

def normalize_domain(domain):
    return domain.strip()

def merge_and_deduplicate(existing_domains, new_domains):
    seen = set()
    result = []
    
    # Process existing domains first to preserve their order and casing
    for d in existing_domains:
        d_norm = normalize_domain(d)
        if not d_norm:
            continue
        d_lower = d_norm.lower()
        if d_lower not in seen:
            seen.add(d_lower)
            result.append(d_norm)
            
    # Process new domains
    for d in new_domains:
        d_norm = normalize_domain(d)
        if not d_norm:
            continue
        d_lower = d_norm.lower()
        if d_lower not in seen:
            seen.add(d_lower)
            result.append(d_norm)
            
    return result

def main():
    parser = argparse.ArgumentParser(
        description="Migrate old site-exclusions lists to the new mode-specific format."
    )
    parser.add_argument(
        "--target-mode",
        choices=["general", "selective"],
        required=True,
        help="Target exclusion mode ('general' or 'selective') to migrate into."
    )
    parser.add_argument(
        "--input",
        action="append",
        help="Path to an old exclusions list file. Can be specified multiple times. If omitted, all files in the legacy exclusions directory (except general.txt and selective.txt) will be scanned."
    )
    
    args = parser.parse_args()
    
    legacy_dir = os.path.expanduser("~/.local/share/adgui/site-exclusions")
    target_dir = os.path.expanduser("~/.config/adgui/site-exclusions")
    target_filename = f"{args.target_mode}.txt"
    target_path = os.path.join(target_dir, target_filename)
    
    input_files = []
    if args.input:
        for f in args.input:
            input_files.append(os.path.abspath(f))
    else:
        # Scan legacy directory
        if not os.path.isdir(legacy_dir):
            print(f"Directory {legacy_dir} does not exist. No files to auto-scan.", file=sys.stderr)
            sys.exit(0)
            
        ignore_files = {"general.txt", "selective.txt"}
        try:
            entries = sorted(os.listdir(legacy_dir))
            for entry in entries:
                entry_path = os.path.join(legacy_dir, entry)
                if os.path.isfile(entry_path) and entry not in ignore_files:
                    input_files.append(entry_path)
        except OSError as e:
            print(f"Error reading directory {legacy_dir}: {e}", file=sys.stderr)
            sys.exit(1)
            
    if not input_files:
        print("No input files specified or found for migration.", file=sys.stderr)
        sys.exit(0)
        
    print(f"Found {len(input_files)} file(s) for migration:")
    for f in input_files:
        print(f"  - {f}")
        
    # Read existing target file if it exists
    existing_domains = []
    if os.path.isfile(target_path):
        try:
            with open(target_path, "r", encoding="utf-8") as f:
                existing_domains = [line.strip() for line in f if line.strip()]
            print(f"Loaded {len(existing_domains)} existing domain(s) from target: {target_path}")
        except OSError as e:
            print(f"Error reading target file {target_path}: {e}", file=sys.stderr)
            sys.exit(1)
            
    # Read new domains
    new_domains = []
    for filepath in input_files:
        if not os.path.isfile(filepath):
            print(f"Warning: File {filepath} does not exist. Skipping.", file=sys.stderr)
            continue
        try:
            with open(filepath, "r", encoding="utf-8") as f:
                count = 0
                for line in f:
                    trimmed = line.strip()
                    if trimmed:
                        new_domains.append(trimmed)
                        count += 1
                print(f"Read {count} domain(s) from {filepath}")
        except OSError as e:
            print(f"Error reading file {filepath}: {e}", file=sys.stderr)
            
    # Merge and deduplicate
    merged = merge_and_deduplicate(existing_domains, new_domains)
    print(f"Merging completed. Total unique domains after merge: {len(merged)}")
    
    # Ensure target directory exists
    try:
        os.makedirs(target_dir, exist_ok=True)
    except OSError as e:
        print(f"Failed to create directory {target_dir}: {e}", file=sys.stderr)
        sys.exit(1)
        
    # Write to target file
    try:
        with open(target_path, "w", encoding="utf-8") as f:
            for domain in merged:
                f.write(domain + "\n")
        print(f"Successfully wrote {len(merged)} domain(s) to {target_path}")
    except OSError as e:
        print(f"Failed to write target file {target_path}: {e}", file=sys.stderr)
        sys.exit(1)

if __name__ == "__main__":
    main()
