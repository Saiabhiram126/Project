"use client"; // ✅ Ensure this is at the top

import { useEffect, useState } from "react";
import axios from "axios";

interface Task {
  id: number;
  title: string;
  status: string;
}

export default function Dashboard() {
  const [tasks, setTasks] = useState<Task[]>([]);
  const [error, setError] = useState<string>("");
  const socket = new WebSocket("ws://localhost:8080/ws"); 
  useEffect(() => {
    const fetchTasks = async () => {
      try {
        const token = localStorage.getItem("token");
        if (!token) {
          window.location.href = "/login";
          return;
        }

        const res = await axios.get<Task[]>("http://localhost:8080/tasks", {
          headers: { Authorization: `Bearer ${token}` },
        });

        console.log("API Response:", res.data); // ✅ Log response in console
        setTasks(res.data || []); // ✅ Ensure tasks is always an array
      } catch (err) {
        console.error("Failed to fetch tasks", err);
        setError("Failed to fetch tasks. Please try again.");
        setTasks([]); // ✅ Prevent null state
      }
    };

    fetchTasks();

    // ✅ Connect WebSocket for real-time updates
    const socket = new WebSocket("ws://localhost:8080/ws");

    socket.onmessage = (event) => {
      const updatedTask: Task = JSON.parse(event.data);
      setTasks((prevTasks) => [...prevTasks, updatedTask]);
    };

    return () => {
      socket.close();
    };
  }, []); // ✅ Ensure it only runs once

  return (
    <div className="flex flex-col items-center justify-center h-screen bg-gray-100">
      <h1 className="text-2xl font-bold mb-4">Dashboard</h1>

      {error && <p className="text-red-500">{error}</p>}

      <ul className="bg-white shadow-md p-4 rounded w-1/2">
        {tasks.length > 0 ? ( // ✅ Ensure tasks is not null
          tasks.map((task) => (
            <li key={task.id} className="p-2 border-b">
              {task.title} - <span className="text-gray-600">{task.status}</span>
            </li>
          ))
        ) : (
          <p>Loading tasks...</p> // ✅ Show a fallback message instead of breaking
        )}
      </ul>
    </div>
  );
}
