import WebSocket from "ws";
import { UnimplementedOperationError } from "../errors/unimplemented-operation";
import { IAcknowledgementData } from "../operations/acknowledgement";
import { IAliasData } from "../operations/alias";
import { Answer, IAnswerData } from "../operations/answer";
import { Candidate, ICandidateData } from "../operations/candidate";
import { IGoodbyeData } from "../operations/goodbye";
import { IOfferData, Offer } from "../operations/offer";
import {
  ESIGNALING_OPCODES,
  ISignalingOperation,
  TSignalingData,
} from "../operations/operation";
import { Service } from "./service";
import { EventEmitter } from "events";
import { Bind } from "../operations/bind";
import { BindRejectedError } from "../errors/bind-rejected";
import { ShutdownRejectedError } from "../errors/shutdown-rejected";
import { Shutdown } from "../operations/shutdown";
import { v4 } from "uuid";
import { ConnectionRejectedError } from "../errors/connection-rejected";
import { Connect } from "../operations/connect";

export class SignalingClient extends Service {
  private id = "";
  private client?: WebSocket;
  private asyncResolver = new EventEmitter();

  constructor(
    private address: string,
    private reconnectDuration: number,
    private onConnect: () => Promise<void>,
    private onDisconnect: () => Promise<void>,
    private onAcknowledgement: (id: string) => Promise<void>,
    private getOffer: () => Promise<string>,
    private getAnswer: (offer: string) => Promise<string>,
    private onAnswer: (
      offererId: string,
      answererId: string,
      answer: string,
      handleCandidate: (candidate: string) => Promise<void>
    ) => Promise<void>,
    private onCandidate: (
      offererId: string,
      answererId: string,
      candidate: string
    ) => Promise<void>,
    private onGoodbye: (id: string) => Promise<void>,
    private onAlias: (id: string, alias: string, set: boolean) => Promise<void>
  ) {
    super();
  }

  async open() {
    this.client = new WebSocket(this.address);
    this.client.onmessage = async (operation) =>
      await this.handleOperation(await this.receive(operation.data));
    this.client.onerror = async (e) => {
      this.logger.error("WebSocket error", e);

      this.client?.terminate();
    };
    this.client.onopen = async () => await this.handleConnect();
    this.client.onclose = async () => await this.handleDisconnect();

    this.logger.info("Server connected", { address: this.address });
  }

  async bind(alias: string) {
    this.logger.info("Binding", { id: this.id, alias });

    return new Promise(async (res, rej) => {
      this.asyncResolver.once(
        this.getAliasKey(this.id, alias),
        (set: boolean) =>
          set
            ? res()
            : rej(
                new BindRejectedError(this.getAliasKey(this.id, alias)).message
              )
      );

      await this.send(this.client, new Bind({ id: this.id, alias }));
    });
  }

  async shutdown(alias: string) {
    this.logger.info("Shutting down", { id: this.id, alias });

    return new Promise(async (res, rej) => {
      this.asyncResolver.once(
        this.getAliasKey(this.id, alias),
        (set: boolean) =>
          set
            ? rej(
                new ShutdownRejectedError(this.getAliasKey(this.id, alias))
                  .message
              )
            : res()
      );

      await this.send(this.client, new Shutdown({ id: this.id, alias }));
    });
  }

  async connect(remoteAlias: string) {
    this.logger.info("Connecting", { id: this.id, remoteAlias });

    const clientConnectionId = v4();

    const clientAlias = await new Promise(async (res, rej) => {
      let i = 0;

      this.asyncResolver.on(
        this.getConnectionKey(clientConnectionId),
        (set: boolean) => {
          if (set) {
            i = i + 1;

            if (i >= 2) {
              res();
            }
          } else {
            rej(
              new ConnectionRejectedError(
                this.getConnectionKey(clientConnectionId)
              ).message
            );
          }
        }
      );

      await this.send(
        this.client,
        new Connect({ id: this.id, clientConnectionId, remoteAlias })
      );

      // TODO: Resolve with client alias
    });

    return clientAlias;
  }

  private async handleConnect() {
    this.logger.info("Server connected", { address: this.address });

    await this.onConnect();
  }

  private async handleDisconnect() {
    this.logger.info("Server disconnected", {
      address: this.address,
      reconnectingIn: this.reconnectDuration,
    });

    await this.onDisconnect();

    await new Promise((res) => setTimeout(res, this.reconnectDuration));

    await this.open();
  }

  private async sendCandidate(candidate: Candidate) {
    await this.send(this.client, candidate);

    this.logger.info("Sent candidate", candidate);
  }

  private async handleOperation(
    operation: ISignalingOperation<TSignalingData>
  ) {
    this.logger.debug("Handling operation", operation);

    switch (operation.opcode) {
      case ESIGNALING_OPCODES.GOODBYE: {
        const data = operation.data as IGoodbyeData;

        this.logger.info("Received goodbye", data);

        await this.onGoodbye(data.id);

        break;
      }

      case ESIGNALING_OPCODES.ACKNOWLEDGED: {
        this.id = (operation.data as IAcknowledgementData).id;

        this.logger.info("Received acknowledgement", { id: this.id });

        await this.onAcknowledgement(this.id);

        const offer = await this.getOffer();

        await this.send(
          this.client,
          new Offer({
            id: this.id,
            offer,
          })
        );

        this.logger.info("Sent offer", { id: this.id, offer });

        break;
      }

      case ESIGNALING_OPCODES.OFFER: {
        const data = operation.data as IOfferData;

        this.logger.info("Received offer", data);

        const answer = await this.getAnswer(data.offer);

        await this.send(
          this.client,
          new Answer({
            offererId: data.id,
            answererId: this.id,
            answer,
          })
        );

        this.logger.info("Sent answer", {
          offererId: data.id,
          answererId: this.id,
          answer,
        });

        break;
      }

      case ESIGNALING_OPCODES.ANSWER: {
        const data = operation.data as IAnswerData;

        this.logger.info("Received answer", data);

        await this.onAnswer(
          data.offererId,
          data.answererId,
          data.answer,
          async (candidate: string) => {
            await this.sendCandidate(
              new Candidate({
                offererId: data.offererId,
                answererId: data.answererId,
                candidate,
              })
            );

            this.logger.info("Sent candidate", data);
          }
        );

        break;
      }

      case ESIGNALING_OPCODES.CANDIDATE: {
        const data = operation.data as ICandidateData;

        this.logger.info("Received candidate", data);

        await this.onCandidate(data.offererId, data.answererId, data.candidate);

        break;
      }

      case ESIGNALING_OPCODES.ALIAS: {
        const data = operation.data as IAliasData;

        this.logger.info("Received alias", data);

        if (data.clientConnectionId) {
          await this.notifyConnect(data.set, data.clientConnectionId);
          await this.onAlias(data.id, data.alias, data.set);
        } else {
          await this.notifyBindAndShutdown(data.id, data.alias, data.set);
          await this.onAlias(data.id, data.alias, data.set);
        }

        break;
      }

      default: {
        throw new UnimplementedOperationError(operation.opcode);
      }
    }
  }

  private async notifyConnect(set: boolean, clientConnectionId: string) {
    this.asyncResolver.emit(this.getConnectionKey(clientConnectionId), set);
  }

  private async notifyBindAndShutdown(id: string, alias: string, set: boolean) {
    this.asyncResolver.emit(this.getAliasKey(id, alias), set);
  }

  private getAliasKey(id: string, alias: string) {
    return `alias id=${id} alias=${alias}`;
  }

  private getConnectionKey(clientConnectionId: string) {
    return `connection id=${clientConnectionId}`;
  }
}