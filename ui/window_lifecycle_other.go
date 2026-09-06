// Copyright (C) 2026 Alexander Grafov <grafov@inet.name>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

//go:build !wayland

package ui

// reuseHiddenWindows is true on X11: Close() sets GLFW's closing flag while
// mouse events can still arrive, and Fyne 2.7.4 processMouseMoved lacks an
// isClosing() guard. Hide() keeps the window alive for the next Show().
const reuseHiddenWindows = true
