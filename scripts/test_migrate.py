#!/usr/bin/env python3
import os
import tempfile
import unittest
import shutil
import subprocess

class TestExclusionsMigration(unittest.TestCase):
    def setUp(self):
        self.temp_dir = tempfile.mkdtemp()
        self.old_home = os.environ.get("HOME")
        os.environ["HOME"] = self.temp_dir
        
    def tearDown(self):
        if self.old_home:
            os.environ["HOME"] = self.old_home
        else:
            os.environ.pop("HOME", None)
        shutil.rmtree(self.temp_dir)
        
    def test_migration_logic(self):
        # Create site-exclusions dir
        exclusions_dir = os.path.join(self.temp_dir, ".local", "share", "adgui", "site-exclusions")
        os.makedirs(exclusions_dir, exist_ok=True)
        
        # Create some old files
        old_file_1 = os.path.join(exclusions_dir, "old1.txt")
        with open(old_file_1, "w", encoding="utf-8") as f:
            f.write("example.com\n  GITHUB.COM \n\n")
            
        old_file_2 = os.path.join(exclusions_dir, "old2.txt")
        with open(old_file_2, "w", encoding="utf-8") as f:
            f.write("github.com\nreddit.com\n")
            
        # Create existing target file
        target_file = os.path.join(exclusions_dir, "general.txt")
        with open(target_file, "w", encoding="utf-8") as f:
            f.write("existing.com\n")
            
        # Run migration script via subprocess
        script_path = os.path.abspath(os.path.join(os.path.dirname(__file__), "migrate-site-exclusions.py"))
        result = subprocess.run(
            ["python3", script_path, "--target-mode", "general"],
            capture_output=True,
            text=True
        )
        self.assertEqual(result.returncode, 0)
        
        # Verify result
        self.assertTrue(os.path.isfile(target_file))
        with open(target_file, "r", encoding="utf-8") as f:
            lines = [line.strip() for line in f if line.strip()]
            
        # Should be: existing.com, example.com, GITHUB.COM (case preserved from first), reddit.com
        self.assertEqual(lines, ["existing.com", "example.com", "GITHUB.COM", "reddit.com"])

if __name__ == "__main__":
    unittest.main()
