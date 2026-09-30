// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

import { Composition } from "remotion";
import { Demo, duration, fps, height, width } from "./Demo";

export const Root = () => (
  <Composition id="Demo" component={Demo} durationInFrames={duration} fps={fps} width={width} height={height} />
);
