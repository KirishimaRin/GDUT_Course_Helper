#!/bin/sh
pid=$(lsof -ti:32555 2>/dev/null)
if [ -n "$pid" ]; then
    kill "$pid"
    echo "Course Helper has been stopped."
else
    echo "Course Helper is not running."
fi
