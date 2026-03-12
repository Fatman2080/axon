import React, { Suspense, lazy } from "react";
import { BrowserRouter as Router, Routes, Route } from "react-router-dom";
import Layout from "./components/layout/Layout";
import Home from "./pages/Home";
import Strategies from "./pages/Strategies";
import StrategyDetail from "./pages/StrategyDetail";
import Agents from "./pages/Agents";
import Profile from "./pages/Profile";
import SubmitAgent from "./pages/SubmitAgent";
import XAuthCallback from "./pages/XAuthCallback";
import Docs from "./pages/Docs";
import Points from "./pages/Points";
import Terms from "./pages/Terms";
import Privacy from "./pages/Privacy";

const AgentArenaLab = lazy(
  () => import("./features/agent-arena/pages/AgentArenaLab"),
);
const AgentArena = lazy(
  () => import("./features/agent-arena/pages/AgentArena"),
);

function App() {
  return (
    <Router>
      <Layout>
        <Routes>
          <Route path="/" element={<Home />} />
          <Route path="/strategies" element={<Strategies />} />
          <Route path="/strategies/:id" element={<StrategyDetail />} />
          <Route path="/vault" element={<Agents />} />
          <Route path="/submit-agent" element={<SubmitAgent />} />
          <Route path="/profile" element={<Profile />} />
          <Route path="/profile/:profileId" element={<Profile />} />
          <Route path="/docs" element={<Docs />} />
          <Route path="/points" element={<Points />} />
          <Route path="/terms" element={<Terms />} />
          <Route path="/privacy" element={<Privacy />} />
          <Route path="/auth/x/callback" element={<XAuthCallback />} />
          <Route
            path="/agent-arena"
            element={
              <Suspense
                fallback={
                  <div className="px-6 py-20 text-center text-sm text-white/60">
                    Loading arena...
                  </div>
                }
              >
                <AgentArena />
              </Suspense>
            }
          />
          <Route
            path="/lab/agent-arena"
            element={
              <Suspense
                fallback={
                  <div className="px-6 py-20 text-center text-sm text-white/60">
                    Loading lab...
                  </div>
                }
              >
                <AgentArenaLab />
              </Suspense>
            }
          />
        </Routes>
      </Layout>
    </Router>
  );
}

export default App;
