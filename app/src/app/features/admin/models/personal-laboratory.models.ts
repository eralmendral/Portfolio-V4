export interface LaboratoryTool {
  readonly name: string;
  readonly role: string;
}

export interface Laboratory {
  readonly id: string;
  readonly name: string;
  readonly code: string;
  readonly state: string;
  readonly color: string;
  readonly x: number;
  readonly y: number;
  readonly description: string;
  readonly mission: string;
  readonly purpose: string;
  readonly areas: readonly string[];
  readonly inputs: readonly string[];
  readonly outputs: readonly string[];
  readonly tools: readonly LaboratoryTool[];
  readonly projects: readonly string[];
  readonly future: string;
  readonly connection: string;
}

export type LaboratoryConnection = readonly [sourceId: string, targetId: string];
