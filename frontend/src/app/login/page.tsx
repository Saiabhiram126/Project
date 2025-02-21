"use client";

import { useState } from "react";
import axios from "axios";

export default function Login() {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");

  const handleLogin = async () => {
    try {
      const res = await axios.post("http://localhost:8080/login", { username, password });
      localStorage.setItem("token", res.data.token);
      window.location.href = "/dashboard"; // Redirect to dashboard
    } catch (error) {
      console.error("Login failed", error);
    }
  };

  return (
    <div className="flex flex-col items-center justify-center h-screen bg-gray-100">
      <input className="p-2 m-2 border" placeholder="Username" onChange={(e) => setUsername(e.target.value)} />
      <input className="p-2 m-2 border" type="password" placeholder="Password" onChange={(e) => setPassword(e.target.value)} />
      <button className="p-2 bg-blue-500 text-black" onClick={handleLogin}>Login</button>
    </div>
  );
}
