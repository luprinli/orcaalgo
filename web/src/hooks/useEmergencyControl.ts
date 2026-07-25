import { useState } from "react";
import { risk } from "../api/client";

interface UseEmergencyControlOptions {
  onSuccess?: () => void;
}

export function useEmergencyControl({ onSuccess }: UseEmergencyControlOptions = {}) {
  const [twoFACode, setTwoFACode] = useState("");
  const [show2FA, setShow2FA] = useState<"stop" | "resume" | null>(null);
  const [loading, setLoading] = useState(false);
  const [msg, setMsg] = useState("");

  const trigger2FA = (action: "stop" | "resume") => {
    setTwoFACode("");
    setMsg("");
    setShow2FA(action);
  };

  const cancel2FA = () => {
    setShow2FA(null);
    setTwoFACode("");
  };

  const execute = async () => {
    if (!show2FA || twoFACode.length < 6) return;
    setLoading(true);
    setMsg("");
    try {
      if (show2FA === "stop") {
        const res = await risk.emergencyStop(twoFACode);
        setMsg(res.halted ? "Trading halted. All orders cancelled, positions closed." : "Stop failed. Verify your code.");
      } else {
        const res = await risk.emergencyResume(twoFACode);
        setMsg(res.halted ? "Trading resumed." : "Resume failed. Verify your code.");
      }
      setShow2FA(null);
      setTwoFACode("");
      onSuccess?.();
    } catch (err) {
      setMsg(err instanceof Error ? err.message : "Emergency action failed");
    } finally {
      setLoading(false);
    }
  };

  return {
    twoFACode,
    setTwoFACode,
    show2FA,
    loading,
    msg,
    setMsg,
    trigger2FA,
    cancel2FA,
    execute,
  };
}
