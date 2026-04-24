import "./index.css";
import { Composition } from "remotion";
import { WhipFlowIntro } from "./WhipFlowIntro";

export const RemotionRoot: React.FC = () => {
  return (
    <>
      <Composition
        id="WhipFlowIntro"
        component={WhipFlowIntro}
        durationInFrames={540}
        fps={30}
        width={1920}
        height={1080}
      />
    </>
  );
};
