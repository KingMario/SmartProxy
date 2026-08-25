export type NetworkInterface = {
  index: number;
  name: string;
};

export type ConfigResponse = {
  autoStart: boolean;
  autoUpdateGfwList: boolean;
  bypassDomains: string[];
  companyDomains: string[];
  companyIface: string;
  defaultIface: string;
  extraGfwDomains: string[];
  gfwIface: string;
  gfwProxy: string;
  gfwlistUrl: string;
  httpProxyIface: string;
  port: number;
  verboseLog: boolean;
};

export type FormState = {
  autoStart: boolean;
  autoUpdateGfwList: boolean;
  bypassDomains: string[];
  companyDomains: string[];
  companyIface: string;
  defaultIface: string;
  extraGfwDomains: string[];
  gfwIface: string;
  gfwlistUrl: string;
  httpProxyIface: string;
  port: string;
  verboseLog: boolean;
};

export type StatusResponse = {
  httpProxyIface?: string;
  httpProxyPort?: number;
  logs?: string[];
  port?: number;
  running: boolean;
};

export type DetectResponse = {
  iface?: string;
};

export type SystemProxyResponse = {
  proxy?: string;
};

export type ToastData = {
  kind: "error" | "success";
  message: string;
};

export type ToastState = ToastData | null;

export type ControlAction = "restart" | "start" | "stop";

export type DetectTarget = "company" | "gfw";
